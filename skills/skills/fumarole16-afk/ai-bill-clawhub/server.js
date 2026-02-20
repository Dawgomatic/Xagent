const express = require('express');
const path = require('path');

const app = express();
const PORT = 8003;

// Serve static files from dist directory
app.use(express.static(path.join(__dirname, 'dist')));

// Main billing page
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'dist', 'index.html'));
});

// API endpoint for usage data
app.get('/api/usage', (req, res) => {
  try {
    const usageData = require('./dist/usage.json');
    res.json({
      status: 'success',
      timestamp: new Date().toISOString(),
      data: usageData
    });
  } catch (error) {
    res.json({
      status: 'error',
      message: 'Usage data not available',
      timestamp: new Date().toISOString()
    });
  }
});

// Health check endpoint
app.get('/api/health', (req, res) => {
  res.json({
    status: 'online',
    service: 'ai-bill',
    port: PORT,
    timestamp: new Date().toISOString()
  });
});

// Redirect from old path
app.get('/usage_live.json', (req, res) => {
  res.redirect('/dist/usage.json');
});

app.listen(PORT, () => {
  console.log(`AI Billing System running on port ${PORT}.`);
});