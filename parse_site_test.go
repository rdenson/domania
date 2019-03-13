package main
import (
  "net/http"
  "strings"
  "testing"
)

type mockRedirect struct {
  response *http.Response
  site string
}

func CreateTest(url string, httpStatus int, locationIsHTTPS bool) *mockRedirect {
  testCase := new(mockRedirect)
  fakeRequest, _ := http.NewRequest("GET", url, nil)

  testCase.site = url
  testCase.response = &http.Response{
    Header: make(http.Header),
    Request: fakeRequest,
    StatusCode: httpStatus,
  }

  if locationIsHTTPS {
    testCase.response.Header.Add("Location", strings.Replace(testCase.site, "http", "https", 1))
  }

  return testCase
}

func TestCheckForRedirection(t *testing.T) {
  //test cases
  //  1: redirects to https url
  //  2: redirects to non-secure url
  //  2: no redirection and site requested is unchanged
  var expectedSite = "http://isthisthesame.net"
  tc1 := CreateTest("http://somedomain.com", 301, true)
  tc2 := CreateTest("http://somedomain.com", 302, false)
  tc3 := CreateTest(expectedSite, 200, false)

  found, locIsHTTP, loc := CheckForRedirection(tc1.response)
  if !found || !locIsHTTP || loc[0:8] != "https://" {
    t.Errorf("tc1 - expected to find a redirect to https url> redirected: %t, headerLocation: %s", found, loc)
  }

  found, locIsHTTP, loc = CheckForRedirection(tc2.response)
  if !found || locIsHTTP || loc[0:7] != "http://" {
    t.Errorf("tc2 - expected to find a redirect to http url> redirected: %t, headerLocation: %s", found, loc)
  }

  found, locIsHTTP, loc = CheckForRedirection(tc3.response)
  if found || locIsHTTP || loc != expectedSite {
    t.Errorf("tc3 - expected not to find a redirect, site should remain the same> redirected: %t, headerLocation: %s", found, loc)
  }
}

func TestFormatUrl(t *testing.T) {
  //test cases
  //  1: returns url with http scheme
  //  2: returns url with https scheme
  //  3: returns empty string on url.Parse() error
  //  4: returns empty string on zero length host argument
  //  5: returns url with http scheme if secure argument is false
  tc1 := FormatUrl("host.com", false)
  tc2 := FormatUrl("host.com", true)
  tc3 := FormatUrl("://host.com", false)
  tc4 := FormatUrl("", false)
  tc5 := FormatUrl("https://host.com", false)

  if len(tc1) == 0 || tc1[0:7] != "http://" {
    t.Errorf("tc1 - expected \"http://host.com\", received: \"%s\"", tc1)
  }

  if len(tc2) == 0 || tc2[0:8] != "https://" {
    t.Errorf("tc2 - expected \"http://host.com\", received: \"%s\"", tc2)
  }

  if len(tc3) != 0 {
    t.Error("tc3 - expected empty string on url.Parse() error")
  }

  if len(tc4) != 0 {
    t.Error("tc4 - expected empty string on empty host argument")
  }

  if len(tc5) == 0 || tc5[0:7] != "http://" {
    t.Errorf("tc5 - expected \"http\" scheme, received: \"%s\"", tc5)
  }
}
