#!/bin/bash

# Mac Calendar Access via SSH + AppleScript
# Similar pattern to other mac-* skills

set -euo pipefail

# Determine target Mac
if [ "${MAC_TARGET:-}" = "macbook" ]; then
    SSH_HOST="guymackenzie@guy-mac-m1"
else
    SSH_HOST="guym@doclib"  # Default: Mac Mini
fi

ACTION="${1:-help}"

case "$ACTION" in
    "list-calendars")
        ssh "$SSH_HOST" "osascript -e '
        tell application \"Calendar\"
            set calList to {}
            repeat with cal in calendars
                set end of calList to (name of cal & \" [\" & (color of cal) & \"]\")
            end repeat
            return my join(calList, \"\n\")
        end tell
        
        on join(lst, delim)
            set AppleScript'\''s text item delimiters to delim
            set txt to lst as string
            set AppleScript'\''s text item delimiters to \"\"
            return txt
        end join'"
        ;;
        
    "events")
        DAYS="${2:-7}"
        CALENDAR_FILTER="${3:-}"
        
        ssh "$SSH_HOST" "osascript -e '
        tell application \"Calendar\"
            set startDate to (current date)
            set time of startDate to 0
            set endDate to startDate + ($DAYS * days)
            
            set eventList to {}
            repeat with cal in calendars
                if \"$CALENDAR_FILTER\" is \"\" or (name of cal contains \"$CALENDAR_FILTER\") then
                    set calEvents to (every event of cal whose start date ≥ startDate and start date < endDate)
                    repeat with evt in calEvents
                        set eventInfo to (short date string of (start date of evt)) & \" \" & (time string of (start date of evt)) & \" - \" & (summary of evt)
                        if location of evt is not \"\" then
                            set eventInfo to eventInfo & \" @ \" & (location of evt)
                        end if
                        set eventInfo to eventInfo & \" [\" & (name of cal) & \"]\"
                        set end of eventList to eventInfo
                    end repeat
                end if
            end repeat
            
            if length of eventList = 0 then
                return \"No events in next $DAYS days\"
            else
                return my join(eventList, \"\n\")
            end if
        end tell
        
        on join(lst, delim)
            set AppleScript'\''s text item delimiters to delim
            set txt to lst as string
            set AppleScript'\''s text item delimiters to \"\"
            return txt
        end join'"
        ;;
        
    "today")
        ssh "$SSH_HOST" "osascript -e '
        tell application \"Calendar\"
            set startDate to (current date)
            set time of startDate to 0
            set endDate to startDate + 1 * days
            
            set eventList to {}
            repeat with cal in calendars
                set calEvents to (every event of cal whose start date ≥ startDate and start date < endDate)
                repeat with evt in calEvents
                    set eventInfo to (time string of (start date of evt)) & \" - \" & (summary of evt)
                    if location of evt is not \"\" then
                        set eventInfo to eventInfo & \" @ \" & (location of evt)
                    end if
                    set end of eventList to eventInfo
                end repeat
            end repeat
            
            if length of eventList = 0 then
                return \"No events today\"
            else
                return my join(eventList, \"\n\")
            end if
        end tell
        
        on join(lst, delim)
            set AppleScript'\''s text item delimiters to delim
            set txt to lst as string
            set AppleScript'\''s text item delimiters to \"\"
            return txt
        end join'"
        ;;
        
    "tomorrow")
        ssh "$SSH_HOST" "osascript -e '
        tell application \"Calendar\"
            set startDate to (current date) + 1 * days
            set time of startDate to 0
            set endDate to startDate + 1 * days
            
            set eventList to {}
            repeat with cal in calendars
                set calEvents to (every event of cal whose start date ≥ startDate and start date < endDate)
                repeat with evt in calEvents
                    set eventInfo to (time string of (start date of evt)) & \" - \" & (summary of evt)
                    if location of evt is not \"\" then
                        set eventInfo to eventInfo & \" @ \" & (location of evt)
                    end if
                    set end of eventList to eventInfo
                end repeat
            end repeat
            
            if length of eventList = 0 then
                return \"No events tomorrow\"
            else
                return my join(eventList, \"\n\")
            end if
        end tell
        
        on join(lst, delim)
            set AppleScript'\''s text item delimiters to delim
            set txt to lst as string
            set AppleScript'\''s text item delimiters to \"\"
            return txt
        end join'"
        ;;
        
    "this-week")
        # Calculate days until end of week (Sunday)
        DAYS_TO_SUNDAY=$(ssh "$SSH_HOST" "osascript -e '
        set today to (current date)
        set dayOfWeek to (weekday of today) as integer
        set daysToSunday to 8 - dayOfWeek
        return daysToSunday'")
        
        "$0" events "$DAYS_TO_SUNDAY"
        ;;
        
    "search")
        QUERY="${2:-}"
        DAYS="${3:-30}"
        
        if [ -z "$QUERY" ]; then
            echo "Error: Search query required"
            exit 1
        fi
        
        ssh "$SSH_HOST" "osascript -e '
        tell application \"Calendar\"
            set startDate to (current date) - ($DAYS * days)
            set endDate to (current date) + ($DAYS * days)
            
            set eventList to {}
            repeat with cal in calendars
                set calEvents to (every event of cal whose start date ≥ startDate and start date < endDate)
                repeat with evt in calEvents
                    if (summary of evt contains \"$QUERY\") or ((description of evt) contains \"$QUERY\") then
                        set eventInfo to (short date string of (start date of evt)) & \" - \" & (summary of evt)
                        if location of evt is not \"\" then
                            set eventInfo to eventInfo & \" @ \" & (location of evt)
                        end if
                        set eventInfo to eventInfo & \" [\" & (name of cal) & \"]\"
                        set end of eventList to eventInfo
                    end if
                end repeat
            end repeat
            
            if length of eventList = 0 then
                return \"No events found matching: $QUERY\"
            else
                return my join(eventList, \"\n\")
            end if
        end tell
        
        on join(lst, delim)
            set AppleScript'\''s text item delimiters to delim
            set txt to lst as string
            set AppleScript'\''s text item delimiters to \"\"
            return txt
        end join'"
        ;;
        
    "add-event")
        CALENDAR="${2:-}"
        TITLE="${3:-}"
        START_DATE="${4:-}"
        END_DATE="${5:-$START_DATE}"
        LOCATION="${6:-}"
        NOTES="${7:-}"
        
        if [ -z "$CALENDAR" ] || [ -z "$TITLE" ] || [ -z "$START_DATE" ]; then
            echo "Error: calendar, title, and start date required"
            echo "Usage: $0 add-event <calendar> <title> <start-date> [end-date] [location] [notes]"
            exit 1
        fi
        
        ssh "$SSH_HOST" "osascript -e '
        tell application \"Calendar\"
            set targetCalendar to (first calendar whose name is \"$CALENDAR\")
            
            # Parse date (simplified - assumes YYYY-MM-DD or YYYY-MM-DD HH:MM)
            set startDateStr to \"$START_DATE\"
            set endDateStr to \"$END_DATE\"
            
            tell targetCalendar
                set newEvent to make new event with properties {summary:\"$TITLE\", start date:(date startDateStr), end date:(date endDateStr)}
                if \"$LOCATION\" is not \"\" then
                    set location of newEvent to \"$LOCATION\"
                end if
                if \"$NOTES\" is not \"\" then
                    set description of newEvent to \"$NOTES\"
                end if
            end tell
            
            return \"Event created: \" & \"$TITLE\"
        end tell'"
        ;;
        
    "help"|*)
        echo "Mac Calendar - Available actions:"
        echo "  list-calendars          List all calendars"
        echo "  events [days] [cal]     List upcoming events (default: 7 days)"
        echo "  today                   Today's events"
        echo "  tomorrow                Tomorrow's events"
        echo "  this-week               This week's events"
        echo "  search <query> [days]   Search events (default: ±30 days)"
        echo "  add-event <calendar> <title> <start> [end] [location] [notes]"
        echo ""
        echo "Target: $SSH_HOST (use MAC_TARGET=macbook to override)"
        ;;
esac