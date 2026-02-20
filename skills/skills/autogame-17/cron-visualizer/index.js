const fs = require('fs');
const { execSync } = require('child_process');

function parseCronLine(line) {
  const parts = line.trim().split(/\s+/);
  if (parts.length < 6 || line.trim().startsWith('#')) return null;
  
  // Basic parser for standard cron format: m h dom mon dow command
  const schedule = parts.slice(0, 5).join(' ');
  const command = parts.slice(5).join(' ');
  
  return { schedule, command };
}

function expandScheduleToMinutes(schedule) {
  // Simplified expansion for visualization (handles *, */n, lists, ranges)
  // Returns array of minute offsets (0-1439) where the job runs
  const [m, h, dom, mon, dow] = schedule.split(' ');
  const minutes = [];
  
  // Helper to parse field
  const parseField = (field, min, max) => {
    if (field === '*') {
      return Array.from({ length: max - min + 1 }, (_, i) => i + min);
    }
    const values = new Set();
    field.split(',').forEach(part => {
      if (part.includes('/')) {
        const [base, step] = part.split('/');
        const start = base === '*' ? min : parseInt(base);
        for (let i = start; i <= max; i += parseInt(step)) values.add(i);
      } else if (part.includes('-')) {
        const [start, end] = part.split('-');
        for (let i = parseInt(start); i <= parseInt(end); i++) values.add(i);
      } else {
        values.add(parseInt(part));
      }
    });
    return Array.from(values).sort((a, b) => a - b);
  };

  const hoursList = parseField(h, 0, 23);
  const minutesList = parseField(m, 0, 59);

  // Note: DOM/MON/DOW logic omitted for daily visualization simplification
  // We assume it runs "today" for visualization purposes to show potential load
  
  hoursList.forEach(hour => {
    minutesList.forEach(minute => {
      minutes.push(hour * 60 + minute);
    });
  });
  
  return minutes;
}

function generateTimeline(jobs) {
  const timeline = new Array(1440).fill(0); // 1440 minutes in a day
  
  jobs.forEach(job => {
    try {
      const runTimes = expandScheduleToMinutes(job.schedule);
      runTimes.forEach(t => {
        if (t >= 0 && t < 1440) timeline[t]++;
      });
    } catch (e) {
      // Ignore complex schedules we can't parse simply
    }
  });
  
  return timeline;
}

function visualize(timeline) {
  console.log("Cron Schedule Visualization (24h)\n");
  console.log("Time   | Load | Graph");
  console.log("-------|------|--------------------------------------------------");
  
  // Aggregate by hour for clearer display
  for (let h = 0; h < 24; h++) {
    let hourLoad = 0;
    for (let m = 0; m < 60; m++) {
      hourLoad += timeline[h * 60 + m];
    }
    
    // Normalize bar length (max 50 chars)
    const barLen = Math.min(50, hourLoad); 
    const bar = '#'.repeat(barLen);
    const timeStr = `${h.toString().padStart(2, '0')}:00`;
    
    console.log(`${timeStr}  | ${hourLoad.toString().padStart(4)} | ${bar}`);
  }
}

function main() {
  try {
    const output = execSync('crontab -l', { encoding: 'utf8' });
    const lines = output.split('\n');
    const jobs = lines.map(parseCronLine).filter(j => j !== null);
    
    console.log(`Found ${jobs.length} cron jobs.`);
    const timeline = generateTimeline(jobs);
    visualize(timeline);
    
  } catch (error) {
    console.error("Error reading crontab:", error.message);
  }
}

// Export for verification
module.exports = { main, parseCronLine };

if (require.main === module) {
  main();
}
