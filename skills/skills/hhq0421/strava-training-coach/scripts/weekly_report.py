#!/usr/bin/env python3
"""
Weekly Training Report - Sunday summary with trends and recommendations
"""

import os
import json
import urllib.request
from datetime import datetime, timedelta

TOKEN_FILE = os.path.expanduser('~/.strava_tokens.json')
NOTIFICATION_CHANNEL = os.environ.get('NOTIFICATION_CHANNEL', 'discord')

def get_webhook_url():
    if NOTIFICATION_CHANNEL == 'slack':
        return os.environ.get('SLACK_WEBHOOK_URL')
    return os.environ.get('DISCORD_WEBHOOK_URL')

def load_tokens():
    try:
        with open(TOKEN_FILE) as f:
            return json.load(f).get('access_token')
    except:
        return None

def fetch_activities(access_token, days=28):
    after = int((datetime.now() - timedelta(days=days)).timestamp())
    url = f'https://www.strava.com/api/v3/athlete/activities?after={after}&per_page=100'
    req = urllib.request.Request(url, headers={'Authorization': f'Bearer {access_token}'})
    try:
        with urllib.request.urlopen(req) as response:
            return json.loads(response.read().decode())
    except:
        return []

def calculate_weeks(activities):
    """Calculate weekly stats for last 4 weeks"""
    weeks = []
    now = datetime.now()
    
    for i in range(4):
        week_start = now - timedelta(days=now.weekday() + (i * 7))
        week_end = week_start + timedelta(days=7)
        
        week_acts = [a for a in activities 
                     if week_start <= datetime.fromisoformat(a['start_date'].replace('Z', '+00:00')) < week_end]
        
        miles = sum(a.get('distance', 0) for a in week_acts) / 1609.34
        time_mins = sum(a.get('moving_time', 0) for a in week_acts) / 60
        runs = len([a for a in week_acts if a.get('type') == 'Run'])
        
        weeks.append({
            'label': f"Week {-i if i > 0 else 'This'}",
            'miles': miles,
            'time': time_mins,
            'runs': runs
        })
    
    return list(reversed(weeks))

def analyze_intensity_distribution(activities):
    """Calculate easy/moderate/hard distribution"""
    runs = [a for a in activities if a.get('type') == 'Run']
    if not runs:
        return None
    
    easy_hr = int(os.environ.get('MIN_EASY_RUN_HEART_RATE', 145))
    
    easy = len([r for r in runs if r.get('average_heartrate', 0) < easy_hr - 10])
    moderate = len([r for r in runs if easy_hr - 10 <= r.get('average_heartrate', 0) <= easy_hr + 10])
    hard = len([r for r in runs if r.get('average_heartrate', 0) > easy_hr + 10])
    
    total = len(runs)
    return {
        'easy_pct': (easy / total) * 100,
        'moderate_pct': (moderate / total) * 100,
        'hard_pct': (hard / total) * 100
    }

def generate_report(activities):
    """Generate weekly report data"""
    weeks = calculate_weeks(activities)
    intensity = analyze_intensity_distribution(activities)
    
    current_week = weeks[-1] if weeks else {'miles': 0, 'runs': 0}
    prev_week = weeks[-2] if len(weeks) > 1 else {'miles': 0}
    
    # Calculate trend
    trend = "stable"
    if prev_week['miles'] > 0:
        change = ((current_week['miles'] - prev_week['miles']) / prev_week['miles']) * 100
        if change > 10:
            trend = "increasing ⚠️"
        elif change < -10:
            trend = "decreasing"
    
    # 80/20 check
    eight_twenty_ok = intensity and intensity['easy_pct'] >= 75
    
    report = {
        'week_miles': current_week['miles'],
        'week_runs': current_week['runs'],
        'trend': trend,
        'four_week_total': sum(w['miles'] for w in weeks),
        'intensity': intensity,
        'eight_twenty_ok': eight_twenty_ok,
        'weekly_data': weeks
    }
    
    return report

def send_discord_report(report, webhook_url):
    """Send weekly report to Discord"""
    intensity = report['intensity']
    
    fields = [
        {"name": "📊 This Week", "value": f"{report['week_miles']:.1f} mi | {report['week_runs']} runs", "inline": True},
        {"name": "📈 Trend", "value": report['trend'], "inline": True},
        {"name": "🗓️ 4-Week Total", "value": f"{report['four_week_total']:.1f} mi", "inline": True}
    ]
    
    if intensity:
        fields.append({
            "name": "🎯 80/20 Check",
            "value": f"Easy: {intensity['easy_pct']:.0f}% | Moderate: {intensity['moderate_pct']:.0f}% | Hard: {intensity['hard_pct']:.0f}%",
            "inline": False
        })
        
        if report['eight_twenty_ok']:
            fields.append({"name": "✅", "value": "Great job keeping easy days easy!", "inline": False})
        else:
            fields.append({"name": "💡", "value": "Aim for 80% easy runs to build aerobic base.", "inline": False})
    
    # Build mini trend chart
    trend_text = " | ".join([f"{w['label']}: {w['miles']:.1f}mi" for w in report['weekly_data']])
    fields.append({"name": "📉 4-Week Trend", "value": trend_text, "inline": False})
    
    embed = {
        "title": "🏃 Weekly Training Report",
        "color": 0xFC4C02 if report['eight_twenty_ok'] else 0xFFA500,
        "fields": fields,
        "footer": {"text": "Consistency > Intensity"},
        "timestamp": datetime.now().isoformat()
    }
    
    payload = {"embeds": [embed]}
    
    req = urllib.request.Request(
        webhook_url,
        data=json.dumps(payload).encode(),
        headers={'Content-Type': 'application/json'},
        method='POST'
    )
    
    try:
        with urllib.request.urlopen(req):
            print("✅ Weekly report sent")
            return True
    except Exception as e:
        print(f"❌ Failed to send report: {e}")
        return False

def send_slack_report(report, webhook_url):
    """Send weekly report to Slack"""
    intensity = report['intensity']
    
    blocks = [
        {
            "type": "header",
            "text": {"type": "plain_text", "text": "🏃 Weekly Training Report"}
        },
        {
            "type": "section",
            "fields": [
                {"type": "mrkdwn", "text": f"*This Week*\n{report['week_miles']:.1f} mi | {report['week_runs']} runs"},
                {"type": "mrkdwn", "text": f"*Trend*\n{report['trend']}"},
                {"type": "mrkdwn", "text": f"*4-Week Total*\n{report['four_week_total']:.1f} mi"}
            ]
        }
    ]
    
    if intensity:
        eight_twenty_emoji = "✅" if report['eight_twenty_ok'] else "💡"
        blocks.append({
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": f"{eight_twenty_emoji} *80/20 Check*: Easy {intensity['easy_pct']:.0f}% | Moderate {intensity['moderate_pct']:.0f}% | Hard {intensity['hard_pct']:.0f}%"
            }
        })
    
    trend_text = " | ".join([f"{w['label']}: {w['miles']:.1f}mi" for w in report['weekly_data']])
    blocks.append({
        "type": "context",
        "elements": [{"type": "mrkdwn", "text": f"📉 4-Week Trend: {trend_text}"}]
    })
    
    payload = {"blocks": blocks}
    
    req = urllib.request.Request(
        webhook_url,
        data=json.dumps(payload).encode(),
        headers={'Content-Type': 'application/json'},
        method='POST'
    )
    
    try:
        with urllib.request.urlopen(req):
            print("✅ Weekly report sent")
            return True
    except Exception as e:
        print(f"❌ Failed to send report: {e}")
        return False

def main():
    print(f"📊 Generating Weekly Report - {datetime.now().strftime('%Y-%m-%d')}\n")
    
    access_token = load_tokens()
    if not access_token:
        print("❌ No Strava tokens. Run auth.py first.")
        return 1
    
    webhook_url = get_webhook_url()
    if not webhook_url:
        print(f"❌ No webhook URL for {NOTIFICATION_CHANNEL}")
        return 1
    
    activities = fetch_activities(access_token)
    if not activities:
        print("No activities found.")
        return 0
    
    report = generate_report(activities)
    
    if NOTIFICATION_CHANNEL == 'slack':
        send_slack_report(report, webhook_url)
    else:
        send_discord_report(report, webhook_url)
    
    return 0

if __name__ == '__main__':
    exit(main())
