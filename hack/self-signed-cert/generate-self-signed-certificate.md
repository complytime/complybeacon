### 1. Create the new Root CA key

```shell
openssl genrsa -out my-ca.key 2048
```

### 2. Create the new Root CA certificate (This command is now fixed)

```shell
openssl req -x509 -new -nodes -key my-ca.key -sha256 -days 1095 \
-subj "/CN=ComplyBeacon Root CA Test/O=ComplyTime Test Org" \
-out my-ca.crt
```

### 3. Create the server's (compass) private key
```shell
openssl genrsa -out compass.key 2048
```

### 4. Create a Certificate Signing Request (CSR) for the server (compass)
```shell
openssl req -new -key compass.key -out compass.csr -subj "/CN=localhost/O=ComplyTimeCompassTestOrg"
```

### 5. Create an Extension File for SAN (Subject Alternative Name) - Maybe optional
```shell
cat > compass.ext << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
# Replace with the hostname or IP your client will use to connect
DNS.1 = compass
IP.1 = 127.0.0.1
EOF
```

### 6. Use your new Root CA to sign the server's CSR
```shell
openssl x509 -req -in compass.csr -CA my-ca.crt -CAkey my-ca.key -CAcreateserial \
-out compass.crt -days 1095 -sha256 -extfile compass.ext
```

### 7. Create the client's (collector) private key
```shell
openssl genrsa -out collector.key 2048
```

### 8. Create a Certificate Signing Request (CSR) for the server (compass)
```shell
openssl req -new -key collector.key -out collector.csr -subj "/CN=localhost/O=ComplyTimeCollectorTestOrg"
```

### 9. Create an Extension File for SAN (Subject Alternative Name) - Maybe optional
```shell
cat > collector.ext << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
EOF
```

### 10. Use your new Root CA to sign the server's CSR
```shell
openssl x509 -req -in collector.csr -CA my-ca.crt -CAkey my-ca.key -CAcreateserial \
-out collector.crt -days 1095 -sha256 -extfile collector.ext
```

### 11. Clean-up
```shell
rm *.csr *.ext
```
