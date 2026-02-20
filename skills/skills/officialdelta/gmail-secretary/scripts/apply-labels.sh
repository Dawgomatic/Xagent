#!/usr/bin/env bash
# Apply labels from agent classification results
# Input: cache/gmail-triage-labels.json (written by the agent)
# Format: [{"index":0,"id":"...","threadId":"...","label":"...","needsReply":true/false}, ...]
set -euo pipefail

ACCOUNT="${GOG_ACCOUNT:-alan.alwakeel@gmail.com}"
export GOG_KEYRING_PASSWORD="${GOG_KEYRING_PASSWORD:-openclaw}"
LABELS_FILE="/home/delta/.openclaw/workspace/cache/gmail-triage-labels.json"

if [ ! -f "$LABELS_FILE" ]; then
  echo "No labels file found at $LABELS_FILE"
  exit 1
fi

node -e "
const fs=require('fs');
const cp=require('child_process');
const account='$ACCOUNT';
const labels=JSON.parse(fs.readFileSync('$LABELS_FILE','utf8'));

function run(args){
  try{return cp.execFileSync('/home/linuxbrew/.linuxbrew/bin/gog',args,{encoding:'utf8',stdio:['ignore','pipe','pipe']})}catch{return ''}
}

// Ensure labels exist
const LABELS=['Urgent','Needs Reply','Waiting On','Read Later','Receipt / Billing','School','Clubs','Mayo','Admin / Accounts'];
let existing=[];
try{const t=run(['gmail','labels','list','--account',account,'--json']);const o=JSON.parse(t);existing=(o.labels||o||[]).map(x=>x.name).filter(Boolean)}catch{}
for(const name of LABELS){if(!existing.includes(name)){try{run(['gmail','labels','create',name,'--account',account,'--json'])}catch{}}}

// Apply
let applied=0;
for(const l of labels){
  if(!l.threadId||!l.label)continue;
  if(!LABELS.includes(l.label))continue;
  try{run(['gmail','labels','modify',l.threadId,'--add',l.label,'--account',account,'--json']);applied++}catch{}
  if(l.needsReply){try{run(['gmail','labels','modify',l.threadId,'--add','Needs Reply','--account',account,'--json'])}catch{}}
}
console.log('Applied labels to '+applied+' threads');
"
