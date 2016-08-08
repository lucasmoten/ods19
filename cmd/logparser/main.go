package logparser

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"golang.org/x/net/context"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	"github.com/hpcloud/tail"
)

var logs = flag.String("logs", "", "Path to log file to consume")
var dbPath = flag.String("db.path", "logparser.db", "Database path. Will be created.")

/*
 *      __
 *     / /   ____  ____ _____  ____ ______________  _____
 *    / /   / __ \/ __ `/ __ \/ __ `/ ___/ ___/ _ \/ ___/
 *   / /___/ /_/ / /_/ / /_/ / /_/ / /  (__  )  __/ /
 *  /_____/\____/\__, / .___/\__,_/_/  /____/\___/_/
 *            /____/_/
 *
 *  Simple server to parse odrive json logs. Usage:
 *
 *    docker-compose logs --no-color odrive > /path/to/file.log
 *
 *    ./logparser -logs /path/to/file.log
 */

const (
	DBConst = iota
)

func main() {

	flag.Parse()

	db, err := bolt.Open(*dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	PrepareBuckets(db)

	t, err := tail.TailFile(*logs, tail.Config{Follow: true})
	if err != nil {
		log.Fatal(err)
	}

	composePattern := `^(\w+\s+\|)`
	trns := &Transformer{regexp.MustCompile(composePattern)}

	go CollectLogs(db, t, trns)

	svr, err := MakeServer(db)
	if err != nil {
		log.Fatal(err)
	}

	http.ListenAndServe("0.0.0.0:9090", svr)
}

type Transformer struct {
	Pat *regexp.Regexp
}

func (t *Transformer) Transform(s string) string {
	return t.Pat.ReplaceAllString(s, "")
}

func PrepareBuckets(db *bolt.DB) {
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("logs"))
		if err != nil {
			return err
		}
		return nil
	})
}

func CollectLogs(db *bolt.DB, t *tail.Tail, trns *Transformer) {

	createKey := func(sess string, ts int) []byte {
		return []byte(fmt.Sprintf("session!%s!%v", sess, ts))
	}

	for line := range t.Lines {

		cleaned := trns.Transform(line.Text)
		var e Envelope
		err := json.Unmarshal([]byte(cleaned), &e)
		if err != nil {
			continue
		}
		if e.Fields.Session == "" || e.TS == 0 {
			continue
		}
		log.Println("Got Envelope", e)

		// Key format: session!<sessionID>!<ts>
		db.Update(func(tx *bolt.Tx) error {

			b, err := tx.CreateBucketIfNotExists([]byte("logs"))
			if err != nil {
				return err
			}
			err = b.Put(createKey(e.Fields.Session, e.TS), []byte(cleaned))
			if err != nil {
				log.Println("could not put Envelope in bucket ", err)
				return err
			}
			return nil
		})

	}
}

func MakeServer(db *bolt.DB) (http.Handler, error) {

	ctx := context.Background()
	ctx = CtxWithDB(ctx, db)

	r := mux.NewRouter()
	r.Methods("GET").Path("/logs").Handler(&LogsHandler{ctx, AllLogs})

	// TODO session specific method: r.Methods("GET").Path("/logs/{sessionID}").Handler(SessionIDHandler)

	return r, nil
}

type LogsHandler struct {
	Ctx context.Context
	Fn  func(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error)
}

func (h *LogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	code, err := h.Fn(h.Ctx, w, r)
	if err != nil {
		log.Printf("HTTP %d: %q", code, err)
		switch code {
		case http.StatusNotFound:
			http.NotFound(w, r)
		case http.StatusInternalServerError:
			http.Error(w, http.StatusText(code), code)
		default:
			http.Error(w, http.StatusText(code), code)
		}
	}
}

func CtxWithDB(ctx context.Context, db *bolt.DB) context.Context {
	return context.WithValue(ctx, DBConst, db)
}

func DBFromCtx(ctx context.Context) *bolt.DB {
	db, _ := ctx.Value(DBConst).(*bolt.DB)
	return db
}

// AllLogs is a handler that returns all logs in the database.
func AllLogs(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {

	db := DBFromCtx(ctx)
	if db == nil {
		return 500, errors.New("missing db")
	}

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("logs"))

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			w.Write(v)
		}
		return nil

	})

	w.Write([]byte("Done"))

	return 200, nil
}

type Envelope struct {
	Msg    string  `json:"msg"`
	Level  string  `json:"level"`
	TS     int     `json:"ts"`
	Fields Payload `json:"fields"`
}

type Payload struct {
	Session string `json:"session"`
	Node    string `json:"node"`
	Method  string `json:"method"`
	URI     string `json:"uri"`
}
