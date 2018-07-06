
server {
    listen 80;
    listen 443 ssl default_server;
    server_name localhost;
    error_log   /dev/stdout debug;


    ssl_certificate /tmp/docker/pki/server.public;
    ssl_certificate_key /tmp/docker/pki/server.private;
    ssl_client_certificate /tmp/docker/pki/trusted.crt;
    ssl_trusted_certificate /tmp/docker/pki/trusted.crt;
    ssl_verify_depth 10;
    ssl_verify_client on;
    ssl_prefer_server_ciphers on;
    ssl_protocols       TLSv1 TLSv1.1 TLSv1.2;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    proxy_request_buffering off;
    proxy_buffering off;
    client_max_body_size 0;
    underscores_in_headers on;
    # other headers for service
    proxy_pass_request_headers on;

    # SSL configs for connection to client, based on NGINX certificates
    # note: this sets SSL_CLIENT_S_DN header automatically
    proxy_ssl_name twl-server-generic2;
    proxy_ssl_certificate         /tmp/docker/pki/server.public;
    proxy_ssl_certificate_key     /tmp/docker/pki/client.crt;
    proxy_ssl_trusted_certificate /tmp/docker/pki/trusted.crt;
    proxy_ssl_verify_depth  10;
    proxy_ssl_verify        on;
    proxy_ssl_session_reuse on;
    proxy_ssl_protocols           TLSv1 TLSv1.1 TLSv1.2;
    proxy_ssl_ciphers 'ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-SHA384:ECDHE-RSA-AES128-SHA256:EC
DHE-RSA-AES256-SHA:ECDHE-RSA-AES128-SHA:DHE-RSA-AES256-SHA256:DHE-RSA-AES128-SHA256:DHE-RSA-AES256-SHA:DHE-RSA-AES128-SHA:ECDHE-RSA-DES-CBC3-SHA:EDH-RSA-DES-CBC3-SHA:AES256-GCM-SHA384:A
ES128-GCM-SHA256:AES256-SHA256:AES128-SHA256:AES256-SHA:AES128-SHA:DES-CBC3-SHA:HIGH:!aNULL:!eNULL:!EXPORT:!CAMELLIA:!DES:!MD5:!PSK:!RC4';

    set $ssl_client_s_dn_value $ssl_client_s_dn;

    location ^~ /services/object-drive/1.0/ {
        rewrite ^/services/object-drive/1.0/(.*) /$1; 
        set $user_dn_value $ssl_client_s_dn;
        set $external_sys_dn_value '';
        if ($http_user_dn) {
                set $user_dn_value $http_user_dn;
                set $external_sys_dn_value $ssl_client_s_dn_value;
        }
        proxy_set_header EXTERNAL_SYS_DN $external_sys_dn_value;
        proxy_set_header USER_DN $user_dn_value;
        proxy_set_header X-Real-IP  $remote_addr;
        proxy_set_header Host       $host;        
        proxy_pass https://${ODRIVE_SERVICE_HOST}:${ODRIVE_SERVICE_PORT};
        break;
    }




    #
    # Proxy Finder Proxy to docker container
    #
    location ^~ /services/finder/1.0/ {
        rewrite ^/services/finder/1.0/(.*) /$1 break;
        # The command we provide in docker-compose.yml will use 'envsubst' to replace ${VAR} placeholders shown below
        # with actual values.
        proxy_pass https://${FINDER_HOST}:${FINDER_PORT};
        proxy_set_header USER_DN    $ssl_client_s_dn;
        proxy_set_header EXTERNAL_SYS_DN 'cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us';
        proxy_set_header Host       $host;
        proxy_set_header X-Real-IP  $remote_addr;
    }
    location ^~ /services/csx/proxy/1.2/ {
        rewrite ^/services/csx/proxy/1.2/(.*) /$1 break;
        # The command we provide in docker-compose.yml will use 'envsubst' to replace ${VAR} placeholders shown below
        # with actual values.
        proxy_pass https://${FINDER_HOST}:${FINDER_PORT};
        proxy_set_header USER_DN    $ssl_client_s_dn;
        proxy_set_header EXTERNAL_SYS_DN 'cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us';
        proxy_set_header Host       $host;
        proxy_set_header X-Real-IP  $remote_addr;
    }


    #
    # Proxy CTE User Service to docker container
    #
    location ^~ /services/userservice/1.0/ {
        rewrite ^/services/userservice/1.0/(.*) /$1 break;
        # The command we provide in docker-compose.yml will use 'envsubst' to replace ${VAR} placeholders shown below
        # with actual values.
        proxy_pass https://${CTE_USER_SERVICE_HOST}:${CTE_USER_SERVICE_PORT};
        proxy_set_header USER_DN    $ssl_client_s_dn;
        proxy_set_header Host       $host;
        proxy_set_header X-Real-IP  $remote_addr;
    }

    #
    # Proxy aac requests to docker container
    #
    location ^~ /services/aac/1.0/ {
        rewrite ^/services/aac/1.0/(.*) /$1 break;
        # The command we provide in docker-compose.yml will use 'envsubst' to replace ${VAR} placeholders shown below
        # with actual values.
        proxy_pass https://${AAC_SERVICE_HOST}:${AAC_SERVICE_PORT};
        proxy_set_header USER_DN    $ssl_client_s_dn;
        proxy_set_header Host       $host;
        proxy_set_header X-Real-IP  $remote_addr;
    }

    location ^~ /services/aac/1.1/ {
        rewrite ^/services/aac/1.1/(.*) /$1 break;
        # The command we provide in docker-compose.yml will use 'envsubst' to replace ${VAR} placeholders shown below
        # with actual values.
        proxy_pass https://${AAC_SERVICE_HOST}:${AAC_SERVICE_PORT};
        proxy_set_header USER_DN    $ssl_client_s_dn;
        proxy_set_header Host       $host;
        proxy_set_header X-Real-IP  $remote_addr;
    }

    location ^~ /services/aac/1.2/ {
        rewrite ^/services/aac/1.2/(.*) /$1 break;
        # The command we provide in docker-compose.yml will use 'envsubst' to replace ${VAR} placeholders shown below
        # with actual values.
        proxy_pass https://${AAC_SERVICE_HOST}:${AAC_SERVICE_PORT};
        proxy_set_header USER_DN    $ssl_client_s_dn;
        proxy_set_header Host       $host;
        proxy_set_header X-Real-IP  $remote_addr;
    }

    #location /services/apps/1.0 {
    #    expires -1;
    #    proxy_pass https://${CTE_APPS_SERVICE_HOST}:${CTE_APPS_SERVICE_PORT}/services/apps/1.0;
    #    proxy_set_header USER_DN    $ssl_client_s_dn;
    #    proxy_set_header Host       $host;
    #    proxy_set_header X-Real-IP  $remote_addr;
    #}

    location /services/apps/1.0/apps/id/chm_drive {
        try_files $uri /apps/drive/json/chm_drive.json;
    }
    location /piwik/piwik.js {
        try_files $uri /apps/drive/json/piwik.js;
    }
    location /piwik/piwik.php {
        try_files $uri /apps/drive/json/piwik.php;
    }

    location /services/ess/1.0 {
            expires -1;

            # determine USER_DN and EXTERNAL_SYS_DN (if USER_DN exists, impersonation is true)
            set $user_dn_value $ssl_client_s_dn;
            set $external_sys_dn_value '';
            if ($http_user_dn) {
                set $user_dn_value $http_user_dn;
                set $external_sys_dn_value $ssl_client_s_dn_value;
            }

            proxy_set_header EXTERNAL_SYS_DN $external_sys_dn_value;
            proxy_set_header USER_DN $user_dn_value;

            proxy_pass               http://es:9200/services/ess/1.0;
    }


    #
    # csx-saved-search-service requests to docker container
    #
    #location ^~ /services/saved-search/1.0/ {
    #    rewrite ^/services/saved-search/1.0/(.*) /$1 break;
    #    # The command we provide in docker-compose.yml will use 'envsubst' to replace ${VAR} placeholders shown below
    #    # with actual values.
    #    proxy_pass https://${SAVED_SEARCH_SERVICE_HOST}:${SAVED_SEARCH_SERVICE_PORT};
    #    proxy_set_header USER_DN    $ssl_client_s_dn;
    #    proxy_set_header Host       $host;
    #    proxy_set_header X-Real-IP  $remote_addr;
    #}

    #
    # Proxy 2-way SSL connections (i.e., client pki cert) to AWS-based services
    #
    #location ^~ /services/ {
    #    proxy_pass https://bedrock.363-283.io;
    #    proxy_set_header USER_DN $ssl_client_s_dn;
    #    proxy_set_header Host      $host;
    #    proxy_set_header X-Real-IP $remote_addr;
    #}

     #
     # Workaround for jspm_packages living outside the gulp dev package
     #
     location ~ ^/apps/finder/jspm_packages/ {
         root /etc/nginx/apps;
         rewrite ^/apps/finder/jspm_packages/(.*) /jspm_packages/finder/$1 break;
         try_files $uri index.html;
     }

     #
     # Proxy to app finder
     #
     location /apps/finder/ {
         root /opt/;
         try_files $uri /apps/finder/index.html;
     }

     location /apps/drive/ {
         root /opt/;
         try_files $uri /apps/drive/index.html;
     }

    location /acm {
        proxy_pass https://${AAC_SERVICE_HOST}:${AAC_SERVICE_PORT};
        proxy_set_header USER_DN    $ssl_client_s_dn;
        proxy_set_header Host       $host;
        proxy_set_header X-Real-IP  $remote_addr;
    }



#    location / {
#         expires -1;
#         proxy_pass              http://builder:9000;
#         # websocket config for browserSync
#         proxy_http_version 1.1;
#         proxy_set_header Upgrade $http_upgrade;
#         proxy_set_header Connection "upgrade";
#         proxy_set_header Host $host;
#    }


    #
    # Proxy 2-way SSL connections (i.e., client pki cert) to dockerized Piwik analytics service
    #
    #location ^~ /piwik/ {
    #    rewrite ^/piwik/(.*) /$1 break;
    #    proxy_pass      http://piwik-proxy:1280;
    #}

    #
    # Proxy to bedrock-static-assets
    #
    location /static/ {
      root /opt/;
      try_files $uri /dist/pages/404.html;
    }

    #
    # Endpoint for Missing Cert
    #
    location /pkiNotFound {
      root /opt/;
      try_files $uri /static/html/nopkifound.html;
    }

    #
    # Endpoint for Unauthorization
    #
    location /unauthorized/ {
      root /opt/;
      try_files $uri /static/html/unauthorized.html;
    }


    # No PKI Cert from the browser? Display an error page
     error_page 495 496 497 = @no_pki_cert;
     location @no_pki_cert {
       expires -1;
       internal;
       try_files $uri /pkiNotFound;
     }



}
