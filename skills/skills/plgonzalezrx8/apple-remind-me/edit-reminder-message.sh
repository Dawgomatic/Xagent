#!/bin/bash
# Edit a reminder's message/title in Apple Reminders.app
# Usage: edit-reminder-message.sh ID "new message"

ID="$1"
MESSAGE="$2"

# Validate arguments
if [[ -z "$ID" ]]; then
    echo " Error: No reminder ID provided"
    echo "Usage: edit-reminder-message.sh ID \"new message\""
    exit 1
fi

if [[ -z "$MESSAGE" ]]; then
    echo " Error: No message provided"
    echo "Usage: edit-reminder-message.sh ID \"new message\""
    exit 1
fi

# Edit the reminder
OUTPUT=$(remindctl edit "$ID" --title "$MESSAGE" 2>&1)

if [[ $? -eq 0 ]]; then
    echo " Reminder $ID updated to: \"$MESSAGE\""
    echo " Check Reminders.app to see the change"
else
    echo " Error updating reminder: $OUTPUT"
    exit 1
fi
