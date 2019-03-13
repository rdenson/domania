package main

import(
  "flag"
  "fmt"
  "os"
  "strings"

  "github.com/aws/aws-sdk-go/aws"
  "github.com/aws/aws-sdk-go/aws/session"
  "github.com/aws/aws-sdk-go/service/route53"
)


/*
 * customized usage display
 */
func domaniaUsage() {
  preamble := "This program can be operated in one of two modes: automatable and interactive. Automatable mode\n" +
              "relies on arguments to query domains and related information. The results of this query is output\n" +
              "in JSON. Subsequent queries require additional program execution. An interactive mode which prompts\n" +
              "and awaits input from a user. This mode works well for manual exploration within the data.\n\n" +
              "Arguments and decriptions are printed below:"
  fmt.Println(preamble)
  flag.PrintDefaults()
  os.Exit(0)
}

func GetHostedZones(svc *route53.Route53, args *route53.ListHostedZonesInput) ([]*zone, *awsRequest) {
  var resp *route53.ListHostedZonesOutput
  var zones []*zone
  req := new(awsRequest)

  //init request metadata
  req.serviceName = "route53"
  req.serviceFunction = "ListHostedZones"
  req.fatalOnError = true
  //exec api call and handle error
  resp, req.err = svc.ListHostedZones(args)
  req.HandleServiceRequestError()

  zones = make([]*zone, len(resp.HostedZones))
  //hold results in custom struct array
  for i:=0; i<len(resp.HostedZones); i++ {
    currentZone := resp.HostedZones[i]
    currentName := string(*currentZone.Name)[:len(*currentZone.Name)-1]
    z := new(zone)

    //only the last part of the zone id is relevant
    z.id = strings.Split(*currentZone.Id,"/")[2]
    //separate out the domain (eg. example.com -> |example|com|)
    if len(strings.Split(currentName, ".")) > 1 {
      z.domain = strings.Split(currentName, ".")[0]
      z.tld = strings.Split(currentName, ".")[1]
    } else {
      z.domain = currentName
    }

    z.recordCount = *currentZone.ResourceRecordSetCount
    zones[i] = z
  }

  return zones, req
}

func GetRecordsetsForZone(svc *route53.Route53, zoneId string) (*recordset, *awsRequest) {
  var args = &route53.ListResourceRecordSetsInput{
    HostedZoneId: aws.String(zoneId),
    //using the following doesn't work... we'll just filter in memory (*sigh*)
    //StartRecordName: aws.String("*"),
    //StartRecordType: aws.String("A"),
  }
  var moreRecords bool = true
  var resp *route53.ListResourceRecordSetsOutput
  req := new(awsRequest)
  zoneRecordset := make(recordset)

  //init request metadata
  req.serviceName = "route53"
  req.serviceFunction = "ListResourceRecordSets"
  req.fatalOnError = true
  //handle paginated results
  for moreRecords {
    //exec api call and handle error
    resp, req.err = svc.ListResourceRecordSets(args)
    req.HandleServiceRequestError()
    //in-memory filter
    zoneRecordset.HashRecordsetTypes(resp.ResourceRecordSets)
    if resp != nil && *resp.IsTruncated {
      args.SetStartRecordName(*resp.NextRecordName)
      moreRecords = true
    } else {
      moreRecords = false
    }
  }

  //don't pass what could be a massive amount of data, just the reference
  return &zoneRecordset, req
}

/*
 *  custom bubble sort for hosted zones; can sort by the domain or tld field for
 *  an array of hosted zones
 */
func HzSort(domainContainers []*zone, sortTarget string) {
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
  var domainId string
  var moreInput bool = true
  var resourceRecord string
  var userResponse string

  //--- program arguments ---
  //program mode can be: (interactive || automatable)
  autoMode := flag.Bool("a", false, "mode: automatable and silent; use this option for single queries")
  //investigate content served by domains
  checkDomainContent := flag.Bool("c", false, "gathers secure protocol information on content served within a domain")
  flag.StringVar(&domainId, "domain", "", "identifier for a hosted zone")
  flag.StringVar(&resourceRecord, "type", "", "resource record; DNS record type")
  flag.Usage = domaniaUsage
  flag.Parse()

  //initialize access to aws api
  sess := session.Must(session.NewSession())
  route53svc := route53.New(sess)

  //control flow for modes (see arguments)
  if !*autoMode {
    //INTERACTIVE MODE
    fmt.Println("program running in interactive mode")
    fmt.Println("fetching domains (hosted zones)...")
    zones, _ := GetHostedZones(route53svc, &route53.ListHostedZonesInput{})
    HzSort(zones, "domain")
    HzSort(zones, "tld")
    fmt.Printf("found %d domains:\n", len(zones))
    fmt.Println("ID\t\tdomain and recordset count")
    fmt.Println("--------------------------------------------")
    for _, zone := range zones {
      fmt.Printf("%s\t%s, %d records\n", zone.id, zone.DomainToString(), zone.recordCount)
    }

    //user's resource record query loop
    for moreInput {
      //specify domain (hosted zone) id
      fmt.Println("\nyou can lookup resource records but I need a domain ID and record type\nwhich domain? (enter a domain ID)")
      fmt.Scanf("%s", &domainId)
      if strings.ToLower(domainId) == "none" || len(domainId) == 0 {
        fmt.Println("no domain ID specified, exiting")
        os.Exit(0)
      }

      //preempt resource recordsets for specified zone
      zoneRecords, _ := GetRecordsetsForZone(route53svc, domainId)
      //specify resource record type
      fmt.Println("what type of resource record are you looking for?")
      fmt.Printf("choice of: %s\n", strings.Join(zoneRecords.GetDistinctTypes(), ", "))
      fmt.Scanf("%s", &resourceRecord)
      if strings.ToLower(resourceRecord) == "none" || len(resourceRecord) == 0 {
        fmt.Println("no record type specified, exiting")
        os.Exit(0)
      }

      if len((*zoneRecords)[strings.ToUpper(resourceRecord)]) > 0 {
        fmt.Printf("found %d records:\n", len((*zoneRecords)[strings.ToUpper(resourceRecord)]))
        for _, record := range (*zoneRecords)[strings.ToUpper(resourceRecord)] {
          fmt.Printf("%s\n", record.name)
          for _, value := range record.values {
            fmt.Printf("\t%s\n", value)
          }
        }
      } else {
        fmt.Printf("no records found\n")
      }

      fmt.Printf("continue? ")
      fmt.Scanf("%s", &userResponse)
      if strings.ToLower(userResponse) == "no" || strings.ToLower(userResponse) == "n" {
        moreInput = false
        fmt.Println("program exiting")
      }
    }
  } else {
    //AUTOMATABLE MODE
    if *checkDomainContent {
      //--domain content checks
      if len(domainId) > 0 {
        zoneRecords, _ := GetRecordsetsForZone(route53svc, domainId)
        aRecordsForDomain := (*zoneRecords)["A"]
        batch := make(chan string, len(aRecordsForDomain))
        for i:=0; i<len(aRecordsForDomain); i++ {
          go ChanneledParseSite(aRecordsForDomain[i].name, batch)
        }

        fmt.Printf("{\"sites\":[")
        for j:=0; j<len(aRecordsForDomain); j++ {
          fmt.Printf("%s", <- batch)
          if j != len(aRecordsForDomain) - 1 {
            fmt.Printf(",")
          }
        }
        fmt.Printf("]}")
      } else {
        fmt.Println("insufficient arguments, when performing domain content checks:\n" +
                    "\t-domain argument is required")
      }
    } else {
      //--information gathering
      if len(domainId) == 0 {
        zones, _ := GetHostedZones(route53svc, &route53.ListHostedZonesInput{})
        fmt.Println(SerializeZones(zones))
      } else if len(domainId) > 0 && len(resourceRecord) > 0 {
        zoneRecords, _ := GetRecordsetsForZone(route53svc, domainId)
        fmt.Println(zoneRecords.SerializeRecords(resourceRecord))
      } else {
        fmt.Println("insufficient arguments, when information gathering:\n" +
                    "\tno additional arguments: outputs hosted zones\n" +
                    "\t-domain and -type: outputs resource records for a hosted zone")
      }
    }
  }

  //TODO:
  //  test domains, check tls and content served
}
