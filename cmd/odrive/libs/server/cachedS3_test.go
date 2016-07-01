package server_test

import (
	"io"
	"os"
	"testing"

	"decipher.com/object-drive-server/cmd/odrive/libs/server"
	"github.com/uber-go/zap"
)

var testCacheData = `
import java.io.FileInputStream;
import java.security.KeyStore;
import javax.net.ssl.SSLContext;
import javax.net.ssl.KeyManagerFactory;
import javax.net.ssl.TrustManagerFactory;
import java.security.SecureRandom;
import javax.net.ssl.HttpsURLConnection;
import java.net.URL;
import javax.net.ssl.HostnameVerifier;
import javax.net.ssl.SSLSession;
import java.io.DataOutputStream;
import java.io.PrintWriter;
import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.io.OutputStream;
import javax.json.*;

/**
  A Java implementation of the Object-Drive REST API
  
  built like:
  
  
  #!/bin/bash
  endpoint=https://bedrock.363-283.io/services/object-drive/1.0
  endpoint=https://dockervm:8080/services/object-drive/1.0
  classpath=.:javax.json-1.0.4.jar
  javac -classpath $classpath ObjectDriveSDK.java && java -classpath $classpath ObjectDriveSDK \
    $endpoint \
    PKCS12 ../defaultcerts/clients/test_0.p12 password \
    PKCS12 ../defaultcerts/server/truststore.p12 password  #can use JKS
  
 */
public class ObjectDriveSDK {
    KeyStore idStore;
    KeyStore trustStore;
    SSLContext ctx;
    HostnameVerifier hostnameVerifier;
    String apiEndpoint;
    
    /**
      Run it as a main to see the API in action
     */
    public static void main(String[] args) throws Exception {
        //Include initial part of URL, like:
        //
        //   https://bedrock.363-283.io/services/object-drive/1.0
        //
        String apiEndpoint = args[0];
        
        //First arg is the location of the identity
        String IDstoreType = args[1];
        String IDLocation = args[2];
        String IDpass = args[3];
        
        //First arg is the location of the identity
        String TruststoreType = args[4];
        String TrustLocation = args[5];
        String Trustpass = args[6];
        
        //Create end context for invoking endpoints
        ObjectDriveSDK sdk = new ObjectDriveSDK(
            apiEndpoint,
            IDstoreType, IDLocation, IDpass,
            TruststoreType, TrustLocation, Trustpass
        );
        
        
        String fileName = "fark.txt";
        JsonObject acm = Json.createObjectBuilder()
            .add("version","2.1.0")
            .add("classif","U")
            .add("owner_prod",Json.createArrayBuilder().build())
            .add("atom_energy",Json.createArrayBuilder().build())
            .add("sar_id",Json.createArrayBuilder().build())
            .add("sci_ctrls",Json.createArrayBuilder().build())
            .add("disponly_to",Json.createArrayBuilder().add("").build())
            .add("dissem_ctrls",Json.createArrayBuilder().build())
            .add("non_ic",Json.createArrayBuilder().build())
            .add("rel_to",Json.createArrayBuilder().build())
            .add("fgi_open",Json.createArrayBuilder().build())
            .add("fgi_protect",Json.createArrayBuilder().build())
            .add("portion","U")
            .add("banner","UNCLASSIFIED")
            .add("dissem_countries",Json.createArrayBuilder().add("USA").build())
            .add("accms",Json.createArrayBuilder().build())
            .add("macs",Json.createArrayBuilder().build())
            .add("oc_attribs",
                Json.createArrayBuilder()
                    .add(Json.createObjectBuilder()
                        .add("orgs", Json.createArrayBuilder().build())
                        .add("missions", Json.createArrayBuilder().build())
                        .add("regions", Json.createArrayBuilder().build())
                        .build()
                    )           
            )
            .add("f_clearance",Json.createArrayBuilder().add("u").build())
            .add("f_sci_ctrls",Json.createArrayBuilder().build())
            .add("f_accms",Json.createArrayBuilder().build())
            .add("f_oc_org",Json.createArrayBuilder().build())
            .add("f_regions",Json.createArrayBuilder().build())
            .add("f_missions",Json.createArrayBuilder().build())
            .add("f_share",Json.createArrayBuilder().build())
            .add("f_atom_energy",Json.createArrayBuilder().build())
            .add("f_macs",Json.createArrayBuilder().build())
            .add("disp_only","")
            .build();
        JsonObject fileMeta = Json.createObjectBuilder()
            .add("typeName","File")
            .add("name",fileName)
            .add("description","")
            .add("acm",acm.toString())
            .build();
            
        sdk.createObject(
            "fark.txt",
            "text/plain",
            new ByteArrayInputStream(
                fileMeta.toString().getBytes()
            ),
            new ByteArrayInputStream("this is a test".getBytes())
        );
    }
    
    public ObjectDriveSDK(
        String apiEndpoint,
        String IDstoreType, String IDLocation, String IDpass,
        String TruststoreType, String TrustLocation, String Trustpass
    ) throws Exception {
            
        this.apiEndpoint = apiEndpoint;
        
        //Load up a id file
        logInfo("loadID: "+IDstoreType+" "+IDLocation);
        FileInputStream isID = new FileInputStream(IDLocation);
        char[] passID = IDpass.toCharArray();
        loadID(IDstoreType, isID, passID);
        
        //Load up a trust file
        logInfo("loadTrust: "+TruststoreType+" "+TrustLocation);
        FileInputStream isTrust = new FileInputStream(TrustLocation);
        char[] passTrust = Trustpass.toCharArray();
        loadTrust(TruststoreType, isTrust, passTrust);
        
        //Create factories upon factories.
        String alg = KeyManagerFactory.getDefaultAlgorithm();
        KeyManagerFactory kmf = KeyManagerFactory.getInstance(alg);
        kmf.init(idStore, passID);
        TrustManagerFactory tmf = TrustManagerFactory.getInstance(alg);
        tmf.init(trustStore);
        
        //Disable hostname checking.
        hostnameVerifier = new HostnameVerifier() {
            public boolean verify(String hostname, SSLSession session) {
                return true;
            }    
        };
        
        //Got an SSL context
        ctx = SSLContext.getInstance("TLS");
        ctx.init(kmf.getKeyManagers(), tmf.getTrustManagers(), SecureRandom.getInstance("SHA1PRNG"));
        
        //Spawn new connections using this specification.
        HttpsURLConnection.setDefaultSSLSocketFactory(ctx.getSocketFactory());
        HttpsURLConnection.setDefaultHostnameVerifier(hostnameVerifier);
    }
    
    /**
      filename: the filename to give the uploaded file
      metadata: the ObjectMetadata json metadata for the file
      content: the content stream of the file
     */
    public void createObject(String fileName, String mimeType, InputStream metadata, InputStream content) throws Exception {
        //the in-memory buffer for sending bytes
        byte[] buffer = new byte[32*1024];
        
        //Get ready to invoke a URL
        URL url = new URL(apiEndpoint+"/objects");
        HttpsURLConnection con = (HttpsURLConnection)url.openConnection();
        con.setRequestMethod("POST");
        con.setDoOutput(true);
        con.setUseCaches(false);
        con.setDoInput(true);
        
        String boundary = startMultipart(con);
        
        OutputStream os = new DataOutputStream(con.getOutputStream());
                
        sendFieldHead(os,boundary);
        sendField(con,os,"ObjectMetadata","application/json",metadata, buffer, buffer.length);
        sendFieldSeparator(os,boundary);
        sendFile(con,os,"filestream",mimeType,content,buffer,buffer.length, fileName);
        sendFieldSeparator(os,boundary);
        
        logInfo("responseCode:"+con.getResponseCode());        
    }
    
    String startMultipart(HttpsURLConnection con) throws Exception {
        String boundary = "*****"+System.currentTimeMillis() + "*****";
        con.setRequestProperty(
            "Content-Type","multipart/form-data;boundary="+boundary
        );
        return boundary;        
    }
    
    
    void sendFieldHead(OutputStream os,String boundary) throws Exception {
        os.write(("--"+boundary+"\r\n").getBytes());
    }
    
    void sendFieldSeparator(OutputStream os,String boundary) throws Exception {
        os.write(("\r\n--"+boundary+"\r\n").getBytes());
    }

    void sendField(
        HttpsURLConnection con, 
        OutputStream os, 
        String partName, 
        String partType, 
        InputStream is,
        byte[] buffer,
        int bufferLength
    ) throws Exception {
        sendGeneric(con,os,partName,partType,is,buffer,bufferLength,"");
    }

    void sendFile(
        HttpsURLConnection con, 
        OutputStream os, 
        String partName, 
        String partType, 
        InputStream is,
        byte[] buffer,
        int bufferLength,
        String fName
    ) throws Exception {
        sendGeneric(con,os,partName,partType,is,buffer,bufferLength," ; filename=\""+fName+"\"");
    }
    
    void sendGeneric(
        HttpsURLConnection con, 
        OutputStream os, 
        String partName, 
        String partType, 
        InputStream is,
        byte[] buffer,
        int bufferLength,
        String dispTail
    ) throws Exception {
        os.write(("Content-Disposition: form-data; name=\""+partName+"\""+dispTail+"\r\n").getBytes());
        os.write(("Content-Type: "+partType+"\r\n\r\n").getBytes());
        
        os.flush();
        int len = 0;
        int totalLength = 0;
        while( (len = is.read(buffer, 0, bufferLength)) != -1) {
            totalLength += len;
            if( len > 0) {
                os.write(buffer, 0, len);                
            }
        }
    }
    
    void logInfo(String msg) {
        System.out.println(msg);
    }
    
    void loadID(String storeType, FileInputStream is, char[] pass) throws Exception {
        idStore = KeyStore.getInstance(storeType);
        idStore.load(is, pass);
    }
    
    void loadTrust(String storeType, FileInputStream is, char[] pass) throws Exception {
        trustStore = KeyStore.getInstance(storeType);
        trustStore.load(is, pass);
    }
}

`

func TestCacheDrainToSafety(t *testing.T) {
	t.Skip()

	//Setup and teardown
	dirname := "t012345"
	//Create raw cache without starting the purge goroutine
	logger := zap.NewJSON()
	d := server.NewS3DrainProviderRaw(".", dirname, float64(0.50), int64(60*5), float64(0.75), 120, logger)

	//create a small file
	rName := server.FileId("farkFailedInitially")
	uploadedName := d.Resolve(server.NewFileName(rName, ".uploaded"))
	fqUploadedName := d.Files().Resolve(uploadedName)
	//we create the file in uploaded state
	f, err := d.Files().Create(uploadedName)
	if err != nil {
		t.Errorf("Could not create file %s:%v", fqUploadedName, err)
	}

	//cleanup
	defer f.Close()
	defer func() {
		err := d.Files().RemoveAll(server.FileNameCached(dirname))
		if err != nil {
			t.Errorf("Could not remove directory %s:%v", dirname, err)
		}
	}()

	fdata := []byte(testCacheData)
	_, err = f.Write(fdata)

	//Will upload into S3, and cache file
	if _, err = os.Stat("./" + dirname); err == nil {
		fqCachedName := d.Files().Resolve(d.Resolve(server.NewFileName(rName, ".cached")))
		d.DrainUploadedFilesToSafety()
		//Ensure that the file was deleted by being uploaded into S3
		if _, err = os.Stat(fqCachedName); err != nil {
			t.Errorf("should have been cached: %s %v", fqCachedName, err)
		}
	}
}

func TestCacheCreate(t *testing.T) {
	t.Skip()
	//Setup and teardown
	bucket := "decipherers"
	dirname := "t01234"
	//Create raw cache without starting the purge goroutine
	logger := zap.NewJSON()
	d := server.NewS3DrainProviderRaw(".", dirname, float64(0.50), int64(60*5), float64(0.75), 120, logger)

	//create a small file
	rName := server.FileId("fark")
	uploadedName := d.Resolve(server.NewFileName(rName, ".uploaded"))
	fqUploadedName := d.Files().Resolve(uploadedName)
	//we create the file in uploaded state
	f, err := d.Files().Create(uploadedName)
	if err != nil {
		t.Errorf("Could not create file %s:%v", fqUploadedName, err)
	}

	//cleanup
	defer f.Close()
	defer func() {
		err := d.Files().RemoveAll(server.FileNameCached(dirname))
		if err != nil {
			t.Errorf("Could not remove directory %s:%v", dirname, err)
		}
	}()

	fdata := []byte(testCacheData)
	//put bytes into small file
	_, err = f.Write(fdata)
	if err != nil {
		t.Errorf("could not write to %s:%v", fqUploadedName, err)
	}

	//Write it to S3
	err = d.CacheToDrain(&bucket, rName, int64(len(fdata)))
	if err != nil {
		t.Errorf("Could not cache to drain:%v", err)
	}
	//Delete it from cache manually
	cachedName := d.Resolve(server.NewFileName(rName, ".cached"))
	err = d.Files().Remove(cachedName)
	if err != nil {
		t.Errorf("Could not remove cached file:%v", err)
	}

	//See if it is pulled from S3 properly
	herr, err := d.DrainToCache(&bucket, rName)
	if err != nil {
		t.Errorf("Could not drain to cache:%v", err)
	}
	if herr != nil {
		t.Errorf("Could not drain to cache:%v", herr)
	}
	cachingName := d.Resolve(server.NewFileName(rName, ".caching"))
	if _, err = d.Files().Stat(cachingName); os.IsNotExist(err) == false {
		t.Errorf("caching file should be removed:%v", err)
	}
	if _, err = d.Files().Stat(cachedName); os.IsExist(err) {
		t.Errorf("cached file shoud exist:%v", err)
	}

	//Read the file back and verify same content
	f, err = d.Files().Open(cachedName)
	defer f.Close()
	buf := make([]byte, 256)
	lread, err := f.Read(buf)
	if err != nil {
		t.Errorf("unable to read file:%v", err)
	}
	s1 := string(fdata[:lread])
	s2 := string(buf)[:lread]
	if s1 != s2 {
		t.Errorf("content did not come back as same values. %s vs %s", s1, s2)
	}

	totalLength := int64(len(fdata))
	cipherReader, err := d.NewS3Puller(d.Logger, rName, totalLength, 0, -1)
	if err != nil {
		t.Errorf("unable to create puller for S3:%v", err)
	}
	for {
		_, err := cipherReader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("unable to read puller for S3:%v", err)
		}
	}
	cipherReader.Close()
}
