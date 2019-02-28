package main

import(
  "strconv"
  "strings"

  "github.com/aws/aws-sdk-go/service/route53"
)

/*
 *  hosted zone container
 *    holds easy to reference data about found hosted zones
 *    methods below
 */
type hz struct {
  domain string
  id string
  recordCount int64
  tld string
}
//pretty print hz's domain name
func (container *hz) DomainToString() string {
  if len(container.tld) > 0 {
    return container.domain + "." + container.tld
  } else {
    return container.domain
  }
}
//packaging for container contents
func (container *hz) Serialize() string {
  var jsonString strings.Builder

  //there's got to be a better way...
  jsonString.WriteString("{")
  jsonString.WriteString("\"id\":\"" + container.id + "\",")
  jsonString.WriteString("\"domain\":\"" + container.domain + "\",")
  jsonString.WriteString("\"tld\":\"" + container.tld + "\",")
  jsonString.WriteString("\"recordCount\":" + strconv.FormatInt(container.recordCount, 10))
  jsonString.WriteString("}")

  return jsonString.String()
}


/*
 *  resource recordset container
 *    recordset => name/value pair (value is an array)
 *    methods below
 */
type rs struct {
  name string
  isAlias bool
  hzRef string
  values []string
}
//packaging for container contents
func (container *rs) Serialize() string {
  var jsonString strings.Builder

  jsonString.WriteString("{")
  jsonString.WriteString("\"name\":\"" + container.name + "\",")
  jsonString.WriteString("\"isAlias\":" + strconv.FormatBool(container.isAlias) + ",")
  //don't include if the hzRef field is blank/empty
  if len(container.hzRef) > 0 {
    jsonString.WriteString("\"hostedZoneReference\":\"" + container.hzRef + "\",")
  }

  jsonString.WriteString("\"values\":[\"" + strings.Join(container.values, "\",\"") + "\"]")
  jsonString.WriteString("}")

  return jsonString.String()
}


/*
 *  container for multiple resource recordsets
 *    needs work
 */
type zoneRs struct {
  types map[string][]*rs
}
func (zr *zoneRs) GetDistinctTypes() []string {
  var distinceTypes []string = make([]string, len(zr.types))
  var i int = 0

  for resourceRecordType, _ := range zr.types {
    distinceTypes[i] = resourceRecordType
    i++
  }

  return distinceTypes
}
//using the result returned from the aws api call to list resource recordsets,
//assemble a hash of recordset types
func (zr *zoneRs) HashRecordsetTypes(recordsets []*route53.ResourceRecordSet) {
  //for each recordset returned...
  for _, recordset := range recordsets {
    var recordvals strings.Builder
    currentRecordset := new(rs)

    //lop off the dot at the end of the recordset name
    currentRecordset.name = string(*recordset.Name)[:len(*recordset.Name)-1]
    if len(recordset.ResourceRecords) > 0 {
      //and parse the resource records (values of the recordset) into a []string
      for j, rval := range recordset.ResourceRecords {
        recordvals.WriteString(*rval.Value)
        if j < len(recordset.ResourceRecords) - 1 {
          recordvals.WriteString(",")
        }
      }
    } else {
      //handle an alias target
      currentRecordset.isAlias = true
      recordvals.WriteString(*recordset.AliasTarget.DNSName)
      currentRecordset.hzRef = *recordset.AliasTarget.HostedZoneId
    }

    currentRecordset.values = strings.Split(recordvals.String(), ",")
    //add recordset to the correct type bucket
    zr.types[*recordset.Type] = append(zr.types[*recordset.Type], currentRecordset)
  }
}
