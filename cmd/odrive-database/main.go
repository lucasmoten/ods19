package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	configx "decipher.com/object-drive-server/configx"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/urfave/cli"
)

// defaultConfig holds values suitable for a containerized test db.
var defaultConfig = configx.AppConfiguration{
	DatabaseConnection: configx.DatabaseConfiguration{
		Driver:   "mysql",
		Host:     "127.0.0.1",
		Port:     "3306",
		Schema:   "metadatadb",
		Protocol: "tcp",
		Username: "dbuser",
		Password: "dbPassword",
	},
}

func main() {

	app := cli.NewApp()
	app.Name = "odrive-database"
	app.Usage = "odrive database manager for setup and migrations"

	// Declare flags common to commands, and pass them in Flags below.
	confFlag := cli.StringFlag{
		Name:  "conf",
		Usage: "Path to yaml config",
	}

	force := cli.BoolFlag{
		Name:  "force",
		Usage: "ignore safety checks and initialize drop/recreate of schema",
	}

	rootUser := cli.StringFlag{
		Name:  "rootUser",
		Usage: "user required for schema modification; has default for test ",
		Value: "root",
	}

	rootPassword := cli.StringFlag{
		Name:  "rootPassword",
		Usage: "password required for schema modification; has default for test ",
		Value: "dbRootPassword",
	}

	app.Commands = []cli.Command{
		{
			Name:  "init",
			Usage: "Connect and initialize mysql database",
			Flags: []cli.Flag{confFlag, force, rootPassword, rootUser},
			Action: func(clictx *cli.Context) error {
				fmt.Println("Initializing database.")
				err := initialize(clictx)
				if err != nil {
					log.Fatal(err)
				}
				return nil
			},
		},
		{
			Name:  "status",
			Usage: "Print status for configured database",
			Flags: []cli.Flag{confFlag},
			Action: func(clictx *cli.Context) error {
				fmt.Println("Checking DB status.")
				err := status(clictx)
				if err != nil {
					log.Fatal(err)
				}
				return nil
			},
		},
	}

	// Global flags. Used when no "command" passed. Must be repeated above for commands.
	app.Flags = []cli.Flag{
		confFlag,
	}

	// There is no "default" command. Print help and exit.
	app.Action = func(clictx *cli.Context) error {
		fmt.Printf("Must specify command. Run `%s help` for info", app.Name)
		return nil
	}

	app.Run(os.Args)
}

// initialize creates a new database from scratch. Root creds are required.
func initialize(clictx *cli.Context) error {

	var conf configx.AppConfiguration

	path := clictx.String("conf")
	if path != "" {
		var err error
		conf, err = loadConfig(path)
		if err != nil {
			return err
		}
	} else {
		conf = defaultConfig
	}

	// TODO(cm): overwriting conf is ugly, set RootX creds on new conf struct field instead?
	// This would mean adding OD_DB_ROOT_X environment variables, too.
	conf.DatabaseConnection.Username = clictx.String("rootUser")
	conf.DatabaseConnection.Password = clictx.String("rootPassword")

	fmt.Println("connecting to db")
	db, err := newDBConn(conf.DatabaseConnection)
	if err != nil {
		return fmt.Errorf("could not connect to db: %v\n", err)
	}
	tries := 10
	for i := 0; i < tries; i++ {
		if err := db.Ping(); err != nil {
			fmt.Printf("could not ping db: %v\n", err)
			time.Sleep(2 * time.Second)
		} else {
			fmt.Println("database connection established")
			break
		}
	}
	if err := db.Ping(); err != nil {
		return fmt.Errorf("could not ping db: %v", err)
	}
	defer db.Close()
	force := clictx.Bool("force")
	fmt.Println("force schema creation:", force)

	if !isDBEmpty(db) || !force {
		return errors.New("Database is not empty. Please review which DB you're connecting to or run with --force=true.")
	}
	fmt.Println("DB is ready to receive schema")
	if err := createSchema(db); err != nil {
		return err
	}
	fmt.Println("schema created")
	return nil
}

// status reports on the status of the DB given the credentials provided.
func status(clictx *cli.Context) error {

	var conf configx.AppConfiguration

	path := clictx.String("conf")
	if path != "" {
		var err error
		conf, err = loadConfig(path)
		if err != nil {
			return err
		}
	} else {
		conf = defaultConfig
	}

	db, err := newDBConn(conf.DatabaseConnection)
	if err != nil {
		return fmt.Errorf("could not create db connection: %v\n", err)
	}

	// TODO(cm): we can, potentially, add many summary stats here, e.g. object count
	if !isDBEmpty(db) {
		fmt.Println("database is not empty")
		return nil
	}
	fmt.Println("database is empty")
	return nil
}

// loadConfig wraps the conversion of the cli conf parameter to an absolute path.
func loadConfig(path string) (configx.AppConfiguration, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return configx.AppConfiguration{}, fmt.Errorf("path error: %v\n", err)
	}
	return configx.LoadYAMLConfig(absPath)

}

// newDBConn provides a database connection with the given config. For a root connection,
// set Username and Password directly on the conf.
func newDBConn(conf configx.DatabaseConfiguration) (*sqlx.DB, error) {

	tlsConf, err := newTLSConfig(conf.CAPath, conf.ClientCert, conf.ClientKey)
	if err != nil {
		return nil, fmt.Errorf("could not build tls config: %v\n", err)
	}

	mysql.RegisterTLSConfig("custom", tlsConf)

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?tls=custom&parseTime=true&collation=utf8_unicode_ci",
		conf.Username,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.Schema,
	)
	return sqlx.Open("mysql", dsn)
}

// embeddedTLSConfig creates a tls.Config object from embedded mysql assets.
// Assets are checked in, and then embedded. Requires any build task to run
// go-bindata against the directories to embed.
func embeddedTLSConfig() (*tls.Config, error) {
	trustBytes, err := Asset("../../defaultcerts/client-mysql/trust/ca.pem")
	if err != nil {
		return nil, fmt.Errorf("Error getting embedded CA trust: %v", err)
	}
	trustCertPool := x509.NewCertPool()
	if !trustCertPool.AppendCertsFromPEM(trustBytes) {
		return nil, fmt.Errorf("Error adding embedded CA trust to pool: %v", err)
	}
	certBlock, err := Asset("../../defaultcerts/client-mysql/id/client-cert.pem")
	if err != nil {
		return nil, fmt.Errorf("error getting embedded cert PEM data %v", err)
	}
	keyBlock, err := Asset("../../defaultcerts/client-mysql/id/client-key.pem")
	if err != nil {
		return nil, fmt.Errorf("error getting embedded key PEM data %v", err)
	}
	cert, err := tls.X509KeyPair(certBlock, keyBlock)
	if err != nil {
		return nil, fmt.Errorf("Error parsing embedded cert: %v", err)
	}
	cfg := tls.Config{
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                trustCertPool,
		InsecureSkipVerify:       true,
		ServerName:               "twl-server-generic2",
		PreferServerCipherSuites: true,
	}
	cfg.BuildNameToCertificate()
	return &cfg, nil

}

// newTLSConfig returns a tls.Config object. If all 3 paths are empty, default
// embedded certificates are used.
func newTLSConfig(trustPath, certPath, keyPath string) (*tls.Config, error) {

	// TODO(cm): refactor this so getting tls.Config with assets on path vs.
	// embedded looks nicer.

	if trustPath == "" && certPath == "" && keyPath == "" {
		fmt.Println("Using embedded client certificates")
		return embeddedTLSConfig()
	}

	trustBytes, err := ioutil.ReadFile(trustPath)
	if err != nil {
		return nil, fmt.Errorf("Error parsing CA trust %s: %v", trustPath, err)
	}
	trustCertPool := x509.NewCertPool()
	if !trustCertPool.AppendCertsFromPEM(trustBytes) {
		return nil, fmt.Errorf("Error adding CA trust to pool: %v", err)
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("Error parsing cert: %v", err)
	}
	cfg := tls.Config{
		Certificates:             []tls.Certificate{cert},
		ClientCAs:                trustCertPool,
		InsecureSkipVerify:       true,
		ServerName:               "twl-server-generic2",
		PreferServerCipherSuites: true,
	}
	cfg.BuildNameToCertificate()

	return &cfg, nil

}

// isDBEmpty tries to find table "object". If it exists, the schema is already initialized.
// This function can be enhanced with additional checks for more tables, a migrations table, etc.
func isDBEmpty(db *sqlx.DB) bool {

	fmt.Println("performing schema check")
	tx := db.MustBegin()

	var name []string
	stmt := `select table_name from information_schema.tables where table_name = 'object'`
	err := tx.Select(&name, stmt)
	if err != nil {
		log.Println("could not do query:", err)
		return false
	}
	if len(name) == 0 {
		fmt.Println("db returned no results when querying for expected tables")
		return true
	}
	return name[0] != "object"
}

// execStmt executes a SQL string against a database transaction.
func execStmt(db *sqlx.DB, stmt string) error {
	log.Printf("executing statement: %s\n", stmt)
	results, err := db.Exec(stmt)
	if err != nil {
		return err
	}
	n, err := results.RowsAffected()
	if err != nil {
		return err
	}
	log.Printf("rows affected: %v\n", n)
	return err
}

// execFile splits a SQL file on semicolon (";"), and iteratively executes the commands.
// Splitting is necessary because our DB driver does not support multiple statement execution.
func execFile(db *sqlx.DB, path string) error {

	fmt.Println("executing SQL:", path)
	data, err := Asset(path)
	if err != nil {
		return err
	}
	stringified := string(data)
	commands := strings.Split(stringified, ";")
	total := int64(0)
	for _, cmd := range commands {
		cleaned := strings.TrimSpace(cmd)
		if cleaned == "" {
			continue
		}
		results, err := db.Exec(cleaned)
		if err != nil {
			return err
		}
		n, err := results.RowsAffected()
		if err != nil {
			return err
		}
		total += n
	}
	fmt.Println("total rows affected:", total)

	return nil
}

// execStmtTx executes a SQL string against the provided transaction.
// It is the caller's responsibility to commit or rollback the transaction.
func execStmtTx(tx *sqlx.Tx, stmt string) error {

	_, err := tx.Exec(stmt)
	if err != nil {
		return err
	}

	return nil
}

// declareProc wraps declaration of a stored procedure or function in a file
// as a single transactional statement. There can be no calls to DELIMETER
// in the files, and there can be only one statement per file.
func declareProc(db *sqlx.DB, path string) error {

	tx := db.MustBegin()
	data, err := Asset(path)
	if err != nil {
		return err
	}
	stringified := string(data)
	if err := execStmtTx(tx, stringified); err != nil {
		tx.Rollback()
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
