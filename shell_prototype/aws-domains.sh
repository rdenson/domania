#/bin/bash
OIFS="$IFS"
IFS=$'\n'

evaluateResourceRecords() {
  echo "looking at $recordType records associated to $zone"
  records=$(aws route53 list-resource-record-sets --hosted-zone-id $zone --query "ResourceRecordSets[?Type == '$recordType'].{name:Name,values:ResourceRecords[*].Value}")
  if [ $(echo ${records} | jq length) -eq 0 ]; then
    echo "no $recordType records found"
  else
    if [ "$recordType" = "A" ]; then
      # for A records, let's see if there's a site or something being served
      for row in $(echo "${records}" | jq -c '.[]'); do
        fqdn=$(echo "${row}" | jq -r '.name')
        # remove trailing "."
        echo "  inspecting ${fqdn%?}..."
        # get data about content served over 443
        ./site.sh --uri="${fqdn%?}" --quiet
      done
    else
      # informational
      echo "found values for $recordType record(s)"
      echo ""
      for row in $(echo "${records}" | jq -c '.[].values[]'); do
        echo $(echo "${row}" | jq -r '.')
      done
    fi
  fi

  echo ""
  promptDifferentRecordType
}

promptDifferentRecordType() {
  echo "whould you like to search again?"
  echo "  1 - yes"
  echo "  2 - no"
  read -sr -n1 response
  if [ $response -eq 1 ]; then
    userInput
    evaluateResourceRecords
  else
    echo "exiting..."
    exit 0
  fi
}

# prompts/collects user input for specifying a hosted zone and specific record set
# arguments:
#   - optionally prompt for the hosted zone ID
userInput() {
  # set zone?
  if [ -n "$1" ] && [ "$1" = "true" ]; then
    read -r -p "what zone? " zone
  fi

  read -r -p "recordset type? (defaults to A records) " recordType
  # default to search A records
  if [ -z "$recordType" ]; then
    recordType="A"
  fi
}
# -------


currentUserContext=$(aws sts get-caller-identity)
# display who you are...
echo "executing as $(echo ${currentUserContext} | jq -r '.Arn')"
echo "now showing zones for account: $(echo ${currentUserContext} | jq '.Account')"
# list domains
aws route53 list-hosted-zones --query "HostedZones[*].{id:Id, name:Name, records:ResourceRecordSetCount}" --output table

echo "from the zones listed above, choose an id; you don't need the \"/hostedzone/\" part"
# let the user pick the type of records in a particular zone to check
userInput true
# inspect Route53 with user input
evaluateResourceRecords
