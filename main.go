package main

import(
  "fmt"
  "os"
  "strconv"
  "strings"

  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/route53"
)


func HzSort(domainContainers []*hz, sortTarget string) {
  var sorted bool = false

  for !sorted {
    sorted = true
    for i:=1; i<len(domainContainers); i++ {
      pos := 0
      continueComparison := true
      //let's begin by looking at each struct's sortable field (either "domain" or "tld")
      behind := strings.ToLower(domainContainers[i-1].domain)
      current := strings.ToLower(domainContainers[i].domain)
      if sortTarget == "tld" {
        behind = strings.ToLower(domainContainers[i-1].tld)
        current = strings.ToLower(domainContainers[i].tld)
      }

      //lexicographical comparison; could've used sort.Strings() but, these strings are relatively simple
      for (pos < len(behind) && pos < len(current)) && continueComparison {
        /*
          handle three cases for comparison:
            1) a letter in the previous word is greater than a letter at the same position in the current word;
            change the order of the structs
            2) the letters we're comparing are the same;
            advance to the next letter in the words
            3) the letter in the previous word is less than a letter at the same position in the current word;
            we're already in the correct order
        */
        if behind[pos] > current[pos] {
          temp := domainContainers[i - 1]
          domainContainers[i - 1] = domainContainers[i]
          domainContainers[i] = temp
          sorted = false
          continueComparison = false
        } else if behind[pos] == current[pos] {
          pos++
        } else {
          continueComparison = false
        }
      }//end word comparison
    }
  } //end sort iteration
}

//native container for domains; hz=hostedZone
type hz struct {
  domain string
  id string
  recordCount int64
  tld string
}
func (container *hz) DomainToString() string {
  return container.domain + "." + container.tld
}
func (container *hz) Serialize() string {
  var jsonString strings.Builder

  //there's got to be a better way...
  jsonString.WriteString("{")
  jsonString.WriteString("id:" + container.id + ",")
  jsonString.WriteString("domain:" + container.domain + ",")
  jsonString.WriteString("tld:" + container.tld + ",")
  jsonString.WriteString("recordCount:" + strconv.FormatInt(container.recordCount, 10))
  jsonString.WriteString("}")

  return jsonString.String()
}

func main() {
  var zones []*hz

  //authentication; using ~/.aws/credentials?
  sess := session.Must(session.NewSession())
  //service: Route53 and subsequent call to list zones
  svc := route53.New(sess)
  params := &route53.ListHostedZonesInput{}
  //get teh zones
  resp, err := svc.ListHostedZones(params)
  if err != nil {
    fmt.Fprintf(os.Stderr, "[Error] calling service function...\n%s\n\n", err.Error())
    os.Exit(1)
  }

  //hold results in custom struct
  zones = make([]*hz, len(resp.HostedZones))
  //fmt.Printf("found %d hosted zones\n", len(resp.HostedZones))
  for i:=0; i<len(resp.HostedZones); i++ {
    currentZone := resp.HostedZones[i]
    currentName := string(*currentZone.Name)[:len(*currentZone.Name)-1]
    z := new(hz)
    z.id = strings.Split(*currentZone.Id,"/")[2]
    z.domain = strings.Split(currentName, ".")[0]
    z.tld = strings.Split(currentName, ".")[1]
    z.recordCount = *currentZone.ResourceRecordSetCount
    zones[i] = z
  }

  HzSort(zones, "domain")
  HzSort(zones, "tld")

  /*fmt.Println(zones)
  for _, zone := range zones {
    fmt.Printf("%+v\n", zone)
  }

  fmt.Println(zones[0].DomainToString())
  fmt.Println(zones[18].DomainToString())
  fmt.Println(zones[0].Serialize())*/
  for _, zone := range zones {
    fmt.Printf("%s (%s), %d records\n", zone.DomainToString(), zone.id, zone.recordCount)
  }

  //zone selection/inspection
  params2 := &route53.ListResourceRecordSetsInput{
    HostedZoneId: aws.String("Z1WIVEZO0APGGA"),
    //doesn't work... will just filter in memory (*sigh*)
    //StartRecordName: aws.String("*"),
    //StartRecordType: aws.String("A"),
  }
  resp2, err2 := svc.ListResourceRecordSets(params2)
  if err2 != nil {
    fmt.Fprintf(os.Stderr, "[Error] calling service function...\n%s\n\n", err2.Error())
    os.Exit(1)
  }

  fmt.Printf("%+v\n", *resp2)
}
