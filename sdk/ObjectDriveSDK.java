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
  endpoint=https://meme.dime.di2e.net/services/object-drive/0.0
  endpoint=https://proxier:8080/services/object-drive/0.0
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
        //   https://meme.dime.di2e.net/services/object-drive/0.0
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