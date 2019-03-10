/*
 * experimental codeset, will be worked into this project
 */
package main
import (
  "crypto/sha1"
  "crypto/tls"
  "fmt"
  "net/http"
  "net/url"
  "strconv"
  "strings"
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

func CheckForRedirection(r *http.Response) (bool, bool, string) {
  var hasHttps bool = false
  var site string = r.Request.URL.String()
  var redirectFound bool = false

  if r.StatusCode == 301 || r.StatusCode == 302 {
    redirectFound = true
    redirectHeaders := r.Header
    //is there a "Location" header, does it start with "http"?
    if _, available := redirectHeaders["Location"]; available  && redirectHeaders["Location"][0][0:4] == "http"{
      //let's see if we're redirecting to something secure
      redirectedURL, _ := url.Parse(redirectHeaders["Location"][0])
      site = redirectedURL.String()
      if redirectedURL.Scheme == "https" {
        hasHttps = true
      }
    }
  }

  return redirectFound, hasHttps, site
}

func UseStandardClient() *http.Client {
  tr := &http.Transport{
    DisableCompression: true,
    IdleConnTimeout: 30 * time.Second,
    MaxIdleConns: 10,
  }

  client := &http.Client{
    Transport: tr,
    Timeout: 30 * time.Second,
  }
  client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
    //just return the initial response
    return http.ErrUseLastResponse
  }

  return client
}

func UseInsecureClient() *http.Client {
  tr := &http.Transport{
    DisableCompression: true,
    IdleConnTimeout: 30 * time.Second,
    MaxIdleConns: 10,
    TLSClientConfig: &tls.Config {
      InsecureSkipVerify: true,
    },
  }

  client := &http.Client{
    Transport: tr,
    Timeout: 30 * time.Second,
  }
  client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
    //just return the initial response
    return http.ErrUseLastResponse
  }

  return client
}

func LoadRequest(site string, requestClient *http.Client) (*http.Response, error) {
  if site[0:4] != "http" {
    site = "http://" + site
  }

  if requestClient == nil {
    requestClient = UseStandardClient()
  }

  req, _ := http.NewRequest("GET", site, nil)
  req.Close = true
  return requestClient.Do(req)
}

type requestResult struct {
  callError error
  certExpiration time.Time
  cipherSuite string
  redirects bool
  redirectsToHttps bool
  site string
  tlsVersion string
  responseEncrypted bool
  certFingerprint string

  rawResponse *http.Response
}
func (res *requestResult) AnalyzeTLS() {
  var fingerprint strings.Builder
  if res.rawResponse != nil && res.rawResponse.TLS != nil {
    res.responseEncrypted = true
    res.cipherSuite = cipherSuitesByCode[res.rawResponse.TLS.CipherSuite]
    res.tlsVersion = tlsVersionsByCode[res.rawResponse.TLS.Version]

    itr := 0
    certs := res.rawResponse.TLS.PeerCertificates
    for certs[itr].IsCA {
      itr++
    }

    res.certExpiration = certs[itr].NotAfter
    fpBytes := sha1.Sum(certs[itr].Raw)
    for i:=0; i<len(fpBytes); i++ {
      fingerprint.WriteString(strconv.FormatInt(int64(fpBytes[i]), 16))
      if i < (len(fpBytes) - 1) {
        fingerprint.WriteString(":")
      }
    }

    res.certFingerprint = fingerprint.String()
  }
}
func (res *requestResult) Serialize() string {
  var jsonString strings.Builder
  var refinedStatus string = "unresponsive"

  if res.rawResponse != nil {
    refinedStatus = strconv.Itoa(res.rawResponse.StatusCode)
  }

  jsonString.WriteString("{")
  jsonString.WriteString("\"site\":\"" + res.site + "\",")
  jsonString.WriteString("\"status\":" + refinedStatus + ",")
  jsonString.WriteString("\"redirectsToHttps\":" + strconv.FormatBool(res.redirectsToHttps) + ",")
  if res.responseEncrypted {
    jsonString.WriteString("\"cipherSuite\":\"" + res.cipherSuite + "\",")
    jsonString.WriteString("\"tlsVersion\":\"" + res.tlsVersion + "\",")
    jsonString.WriteString("\"certFingerprint\":\"" + res.certFingerprint + "\",")
    jsonString.WriteString("\"certExpiration\":\"" + res.certExpiration.String() + "\",")
  }

  jsonString.WriteString("\"error\":" + strconv.FormatBool(res.callError != nil))
  jsonString.WriteString("}")

  return jsonString.String()
}

func main() {
  //take in some site so we can see what will be returned without recompilation
  var site string
  fmt.Println("gimmie a url:")
  fmt.Scanf("%s", &site)
  //-----

  rr := new(requestResult)
  rr.site = site
  //make request to site (should be with scheme: http)
  resp, respErr := LoadRequest(rr.site, nil)
  if respErr == nil {
    //let's look for a redirect; gather some data when we check
    rr.redirects, rr.redirectsToHttps, rr.site = CheckForRedirection(resp)
    if !rr.redirectsToHttps {
      //ok, no redirection... let's try hitting https with the site inputted
      rr.site = "https://" + site
    }

    //reload the request, trying to hit https
    resp, respErr = LoadRequest(rr.site, nil)
  }

  //error anticipation
  if respErr != nil && strings.Contains(respErr.Error(), "x509:") {
    //error mentions something about the cert, hit it again and don't try to verify the cert
    fmt.Println("retry and skip certificate verification")
    resp, respErr = LoadRequest(rr.site, UseInsecureClient())
  }

  //set the last response/error now
  rr.rawResponse = resp
  rr.callError = respErr
  rr.AnalyzeTLS()
  fmt.Printf("%s\n", rr.Serialize())
  /*req, _ := http.NewRequest("GET", site, nil)
  req.Close = true
  resp, err := client.Do(req)
  if err != nil {
    fmt.Printf("!!error!!\n%+v\n", err)
    os.Exit(1)
  }

  if err == nil && (resp.StatusCode == 301 || resp.StatusCode == 302) {
    redirectHeaders := resp.Header
    if _, available := redirectHeaders["Location"]; available {
      parsedLocation, _ := url.Parse(redirectHeaders["Location"][0])
      if parsedLocation.Scheme == "https" {
        req.URL = parsedLocation
        resp, err = client.Do(req)
        fmt.Println("<< redirected >>")
      } else {
        fmt.Println("<< does not have https equivalent >>")
      }
    }
  } else {
    fmt.Println("<< no redirection >>")
  }*/

  ////////////////////
  /*fmt.Println("received the following> ")
  fmt.Printf("%s\nstatus: %d\ncontent length received: %d\n\n", resp.Proto, resp.StatusCode, resp.ContentLength)
  fmt.Println("headers:")
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
  }*/
}
