/*
 * experimental codeset, will be worked into this project
 */
package main
import (
  "crypto/sha1"
  "fmt"
  "net/http"
  "os"
  "time"
)


//enumeration of cipher suites
var cipherSuitesByCode = map[uint16]string{
  0x0005: "TLS_RSA_WITH_RC4_128_SHA",
  0x000a: "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
  0x002f: "TLS_RSA_WITH_AES_128_CBC_SHA",
  0x0035: "TLS_RSA_WITH_AES_256_CBC_SHA",
  0x003c: "TLS_RSA_WITH_AES_128_CBC_SHA256",
  0x009c: "TLS_RSA_WITH_AES_128_GCM_SHA256",
  0x009d: "TLS_RSA_WITH_AES_256_GCM_SHA384",
  0xc007: "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
  0xc009: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
  0xc00a: "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
  0xc011: "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
  0xc012: "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
  0xc013: "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
  0xc014: "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
  0xc023: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
  0xc027: "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
  0xc02f: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
  0xc02b: "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
  0xc030: "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
  0xc02c: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
  0xcca8: "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
  0xcca9: "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
  0x1301: "TLS_AES_128_GCM_SHA256",
  0x1302: "TLS_AES_256_GCM_SHA384",
  0x1303: "TLS_CHACHA20_POLY1305_SHA256",
  // TLS_FALLBACK_SCSV isn't a standard cipher suite but an indicator
  // that the client is doing version fallback. See RFC 7507.
  0x5600: "TLS_FALLBACK_SCSV",
}
//enumeration of ssl/tls versions
var tlsVersionsByCode = map[uint16]string{
  0x0300: "VersionSSL30",
  0x0301: "VersionTLS10",
  0x0302: "VersionTLS11",
  0x0303: "VersionTLS12",
  0x0304: "VersionTLS13",
}

func main() {
  tr := &http.Transport{
    MaxIdleConns: 10,
    IdleConnTimeout: 30 * time.Second,
    DisableCompression: true,
  }

  client := &http.Client{Transport: tr}
  //take in some site so we can see what will be returned without recompilation
  var url string
  fmt.Println("gimmie a url:")
  fmt.Scanf("%s", &url)
  //-----
  resp, err := client.Get(url)
  if err != nil {
    fmt.Printf("!!error!!\n%+v\n", err)
    os.Exit(1)
  }

  ////////////////////
  fmt.Println("received the following> ")
  fmt.Printf("%s\nstatus: %d\nheaders:\n", resp.Proto, resp.StatusCode)
  fmt.Printf("content length received: %d\n\n", resp.ContentLength)
  for header, value := range resp.Header {
    fmt.Printf("\t%s: %s\n", header, value)
  }

  if resp.TLS != nil {
    //talk about state of tls connection
    fmt.Printf("site connection encrypted, using:\nversion:\t%s\ncipher:\t\t%s\n", tlsVersionsByCode[resp.TLS.Version], cipherSuitesByCode[resp.TLS.CipherSuite])
    fmt.Printf("handshake completed: %t\n", resp.TLS.HandshakeComplete)
    fmt.Printf("connection presented %d certificate(s)\n", len(resp.TLS.PeerCertificates))
    //look at certs presented
    for i:=0; i<len(resp.TLS.PeerCertificates); i++ {
      currentCert := resp.TLS.PeerCertificates[i]
      //certs are displayed (numbered) with a SHA1 fingerprint then some datapoints
      fmt.Printf("%d) - % x\n\tissuer: %+v\n\tsubject: %+v\n", i+1, sha1.Sum(currentCert.Raw), currentCert.Issuer, currentCert.Subject)
      fmt.Printf("\texpires on %s\n", currentCert.NotAfter.String())
      //subject alternative names
      if len(currentCert.DNSNames) > 0 {
        fmt.Printf("\tSANs-DNS:\n\t\t%v\n", currentCert.DNSNames)
      }
      if len(currentCert.IPAddresses) > 0 {
        fmt.Printf("\tSANs-IPs:\n\t\t%v\n", currentCert.IPAddresses)
      }
      //TODO: maybe a container to hold each cert then do something if it's a CA?
      fmt.Printf("\tCA: %t\n\n", currentCert.IsCA)
    }
  } else {
    fmt.Println("site connection is unencrypted")
  }
}
