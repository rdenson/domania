/*
 * experimental codeset, will be worked into this project
 */
package main
import (
  "crypto/sha1"
  "crypto/tls"
  "net/http"
  "net/url"
  "strconv"
  "strings"
  "time"
)


//enumeration of cipher suites (from GO crypto/tls)
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
//enumeration of ssl/tls versions (from GO crypto/tls)
var tlsVersionsByCode = map[uint16]string{
  0x0300: "VersionSSL30",
  0x0301: "VersionTLS10",
  0x0302: "VersionTLS11",
  0x0303: "VersionTLS12",
  0x0304: "VersionTLS13",
}
var standardTransport *http.Transport = &http.Transport{
  DisableCompression: true,
  IdleConnTimeout: 30 * time.Second,
  MaxIdleConns: 10,
}
var insecureTransport *http.Transport = &http.Transport{
  DisableCompression: true,
  IdleConnTimeout: 30 * time.Second,
  MaxIdleConns: 10,
  TLSClientConfig: &tls.Config {
    InsecureSkipVerify: true,
  },
}
var httpClient *http.Client = &http.Client{
  Transport: standardTransport,
  Timeout: 30 * time.Second,
}

/*
 *  Given an HTTP response, does the request redirect us and does it redirect us
 *  somewhere secure?
 *  This function is relying on an http client that intercepts redirects.
 *  see: http.client -> CheckRedirect()
*/
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

/*
 *  Make sure our requests have a properly formatted URL. This function can force
 *  http or https schemes.
 */
func FormatUrl(host string, secure bool) string {
  formattedUrl, err := url.Parse(host)
  if err != nil || len(host) == 0 {
    //TODO: better logging
    //fmt.Printf("FormatUrl() - could not format: %s\n", host)
    return ""
  }

  if secure {
    formattedUrl.Scheme = "https"
  } else {
    formattedUrl.Scheme = "http"
  }

  return formattedUrl.String()
}

func LoadRequest(site string, insecure bool) (*http.Response, error) {
  //set TLS configuration...
  if insecure {
    httpClient.Transport = insecureTransport
  } else {
    httpClient.Transport = standardTransport
  }

  //intercept the request redirect
  httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
    //just return the initial response
    return http.ErrUseLastResponse
  }

  req, _ := http.NewRequest("GET", site, nil)
  req.Close = true
  return httpClient.Do(req)
}

/*
 *  Container that holds our results after making a request.
 */
type requestResult struct {
  callError error
  certExpiration time.Time
  certFingerprint string
  certIssuer string
  certSubject string
  cipherSuite string
  redirects bool
  redirectsToHttps bool
  responseEncrypted bool
  site string
  tlsVersion string

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
    res.certIssuer = certs[itr].Issuer.String()
    res.certSubject = certs[itr].Subject.String()
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
  var refinedStatus int = -1

  if res.rawResponse != nil {
    refinedStatus = res.rawResponse.StatusCode
  }

  jsonString.WriteString("{")
  jsonString.WriteString("\"site\":\"" + res.site + "\",")
  jsonString.WriteString("\"status\":" + strconv.Itoa(refinedStatus) + ",")
  jsonString.WriteString("\"redirectsToHttps\":" + strconv.FormatBool(res.redirectsToHttps) + ",")
  if res.responseEncrypted {
    //finding some things in the issuer and subject that we need to escape...
    cleanIssuer := strings.Replace(res.certIssuer, "\\", "\\\\", -1)
    cleanSubject := strings.Replace(res.certSubject, "\\", "\\\\", -1)

    jsonString.WriteString("\"cipherSuite\":\"" + res.cipherSuite + "\",")
    jsonString.WriteString("\"tlsVersion\":\"" + res.tlsVersion + "\",")
    //placing cert specific datapoint in a separate object
    jsonString.WriteString("\"cert\":{")
    jsonString.WriteString("\"issuer\":\"" + cleanIssuer + "\",")
    jsonString.WriteString("\"subject\":\"" + cleanSubject + "\",")
    jsonString.WriteString("\"expiration\":\"" + res.certExpiration.String() + "\",")
    jsonString.WriteString("\"fingerprint\":\"" + res.certFingerprint + "\"")
    jsonString.WriteString("},")
  }

  jsonString.WriteString("\"error\":" + strconv.FormatBool(res.callError != nil))
  //if there is an error however, plebeian, include during serialization
  if res.callError != nil {
    jsonString.WriteString(",")
    jsonString.WriteString("\"errorMessage\":\"" + res.callError.Error() + "\"")
  }

  jsonString.WriteString("}")

  return jsonString.String()
}

/*
 *  Get security information from an HTTP GET request. Tries to understand the state
 *  of a site's security by examining redirect behavior and TLS status.
 */
func ParseSite(uri string) string {
  var parseResults *requestResult = new(requestResult)
  var response *http.Response
  var requestError error

  parseResults.site = uri
  //make request to site (should be with scheme: http)
  response, requestError = LoadRequest(FormatUrl(parseResults.site, false), false)
  if requestError == nil {
    //let's look for a redirect and gather some data while we check
    parseResults.redirects, parseResults.redirectsToHttps, parseResults.site = CheckForRedirection(response)
    //reload the request, trying to hit https
    response, requestError = LoadRequest(FormatUrl(parseResults.site, true), false)
  }

  //error anticipation (request could come from one of the calls above)
  if requestError != nil && strings.Contains(requestError.Error(), "x509:") {
    //error mentions something about the cert, hit it again and don't try to verify the cert
    response, requestError = LoadRequest(FormatUrl(parseResults.site, true), true)
  }

  //set the last response/error now, we're finished with requests
  parseResults.rawResponse = response
  parseResults.callError = requestError
  parseResults.AnalyzeTLS()

  return parseResults.Serialize()
}
