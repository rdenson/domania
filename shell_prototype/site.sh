#/bin/bash
LOG_VERBOSE=true
PORT=443
# output variables
RESP="-"
EXPIRATIONDATE="-"
ISSUER="-"
SUBJECTLINE="-"
CERTHASH="-"

# prints an error and exits this script if fatal
# arguments:
#   - message to print
#   - boolean that will help decide if we should exit
handleError() {
  errorIsFatal=$2
  echo ""
  echo "  Error: $1"
  echo ""
  if [ "$errorIsFatal" = "true" ]; then
    exit 1
  fi
}

# prints a debug message
# arguments:
#   - message to print
log() {
  if [ "$LOG_VERBOSE" = "true" ]; then
    echo "  - $1"
  fi
}

printResult() {
  echo "  $URI:$PORT|$RESP|$ISSUER|$SUBJECTLINE|$EXPIRATIONDATE|$CERTHASH"
}

showWelcome() {
  if [ "$LOG_VERBOSE" = "true" ]; then
    echo "  +++ Site Inspection +++"
    echo "  -----------------------"
    echo "  execute this script using *.sh --uri="" [--port=0] [--quiet] (where brackets [] are optional)"
    echo "  utilization/flow:"
    echo "    attempt to connect to the URI"
    echo "    using openssl, we'll grab a site's certificate"
    echo "    a file (tocheck.crt) will be created with the cert from the site"
    echo "    open cert using openssl x509 and get important infos"
    echo ""
  fi
}

showUsage() {
  if [ "$LOG_VERBOSE" = "true" ]; then
    echo "  script can be called with the following arguments:"
    echo "    - uri: the target to check, ex: --uri=www.example.com"
    echo "    - port (optional): the target's port, defaults to 443"
    echo "    - quiet (optional): tells this script to only print out the pipe-delimited results"
    echo ""
  fi
}
# -------


# read the options
options=$(getopt --longoptions uri:,port:,quiet --name "$(basename "$0")" --options "" -- "$@")
if [ $? -ne 0 ]; then
  showUsage
  exit 1
fi
eval set --$options
# extract options and their arguments into variables.
while [[ $# -gt 0 ]]; do
  case "$1" in
    --uri) URI=$2 ; shift 2 ;;
    --port) PORT=$2 ; shift 2 ;;
    --quiet) LOG_VERBOSE=false ; shift ;;
    --) shift ; break ;;
    *) handleError "something went wrong while parsing the options" true ; exit 1 ;;
  esac
done

# site parameter is required
if [ -z "$URI" ]; then
  showUsage
  handleError "some uri/site is required to check, try running the script again with the right arguments" true
fi

showWelcome
showUsage

# remove existing cert we're checking; cleanup
if [ -e tocheck.crt ]; then
  rm tocheck.crt
fi

# let's begin our checks...
log "connecting to $URI at $PORT..."
respRaw="$(curl -ksI --max-time 10 https://$URI | head -n1)"
# did we receive a response?
if [ -z "$respRaw" ]; then
  log "no reponse"
else
  RESP="${respRaw//[$'\t\r\n']}"
  log "responded with: $RESP, now checking SSL"
  # use SNI (server name indication)
  echo | openssl s_client -connect $URI:443 -servername $URI 2>&1 | sed --quiet '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > tocheck.crt
  certSize=$(wc -l tocheck.crt|cut -d ' ' -f1)
  log "wrote $(wc -l tocheck.crt|cut -d ' ' -f1) lines"
  # could we obtain a certificate?
  if [ $certSize -gt 0 ]; then
    EXPIRATIONDATE=$(openssl x509 -in tocheck.crt -text -noout | grep "Validity" -A2 | grep -Po "(?<=Not After\s:\s).*")
    ISSUER=$(openssl x509 -in tocheck.crt -text -noout | grep -Po "(?<=Issuer:\s).*$" | grep -Po "(?<=CN\=).*$")
    SUBJECTLINE=$(openssl x509 -in tocheck.crt -text -noout | grep -Po "(?<=Subject:\s).*$")
    CERTHASH=$(openssl x509 -in tocheck.crt -modulus -noout | openssl md5 | grep -Po "(?<=\(stdin\)\=\s).*")
    log "issuer: $ISSUER"
    log "subject: $SUBJECTLINE"
    log "expiration: $EXPIRATIONDATE"
    log "md5 hash: $CERTHASH"
  else
    log "uh oh, no certificate found using openssl"
  fi
fi

# result artifact, pipe delimited
printResult
