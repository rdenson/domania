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


func GetHostedZones(svc *route53.Route53, args *route53.ListHostedZonesInput) ([]*hz, *awsRequest) {
  var resp *route53.ListHostedZonesOutput
  var zones []*hz
  //custom response metadata container
  req := new(awsRequest)

  //init request metadata
  req.serviceName = "route53"
  req.serviceFunction = "ListHostedZones"
  req.fatalOnError = true
  //exec api call and handle error
  resp, req.err = svc.ListHostedZones(args)
  req.HandleServiceRequestError()

  zones = make([]*hz, len(resp.HostedZones))
  //hold results in custom struct
  for i:=0; i<len(resp.HostedZones); i++ {
    currentZone := resp.HostedZones[i]
    currentName := string(*currentZone.Name)[:len(*currentZone.Name)-1]
    z := new(hz)

    //only the last part of the zone id is relevant
    z.id = strings.Split(*currentZone.Id,"/")[2]
    //separate out the domain (eg. example.com -> |example|com|)
    z.domain = strings.Split(currentName, ".")[0]
    z.tld = strings.Split(currentName, ".")[1]
    z.recordCount = *currentZone.ResourceRecordSetCount
    zones[i] = z
  }

  return zones, req
}

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

//request metadata container
type awsRequest struct {
  serviceName string
  serviceFunction string
  err error
  fatalOnError bool
}
func (req *awsRequest) HandleServiceRequestError() {
  if req.err != nil {
    //spit out the error
    fmt.Fprintf(os.Stderr, "[Error] calling %s service function %s()...\n%s\n\n", req.serviceName, req.serviceFunction, req.err.Error())
    //halt if necessary
    if req.fatalOnError {
      os.Exit(1)
    }
  }
}

func main() {
  //authentication; using ~/.aws/credentials?
  sess := session.Must(session.NewSession())
  route53svc := route53.New(sess)

  zones, _ := GetHostedZones(route53svc, &route53.ListHostedZonesInput{})
  //sort for pretty display
  HzSort(zones, "domain")
  HzSort(zones, "tld")
  for _, zone := range zones {
    fmt.Printf("%s (%s), %d records\n", zone.DomainToString(), zone.id, zone.recordCount)
  }

  //TODO: move to function
  //      read-eval loop?
  //      arguments?
  //zone selection/inspection
  var resp *route53.ListResourceRecordSetsOutput
  var z = aws.String("Z1WIVEZO0APGGA")
  params := &route53.ListResourceRecordSetsInput{
    HostedZoneId: z,
    //doesn't work... will just filter in memory (*sigh*)
    //StartRecordName: aws.String("*"),
    //StartRecordType: aws.String("A"),
  }
  recordsetsRequest := new(awsRequest)
  recordsetsRequest.serviceName = "route53"
  recordsetsRequest.serviceFunction = "ListResourceRecordSets"
  recordsetsRequest.fatalOnError = true
  resp, recordsetsRequest.err = route53svc.ListResourceRecordSets(params)
  recordsetsRequest.HandleServiceRequestError()

  //nfos, prolly not needed
  fmt.Printf("\nfound %d recordsets for %s\n", len(resp.ResourceRecordSets), *z)
  //filter based on recordset type; wanted to do it in ListResourceRecordSetsInput...
  for _, recordset := range resp.ResourceRecordSets {
    if *recordset.Type == "A" {
      var recordvals strings.Builder
      var recordname string = string(*recordset.Name)[:len(*recordset.Name)-1]

      fmt.Printf("%s\n", recordname)
      //"ResourceRecords" is an array; handle multiple values
      for _, record := range recordset.ResourceRecords {
        recordvals.WriteString(*record.Value + "\n")
      }

      fmt.Println(recordvals.String())
    }
  }
}
