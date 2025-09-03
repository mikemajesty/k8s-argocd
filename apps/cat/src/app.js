require('dotenv').config();
const express = require('express');
const app = express();
const port = process.env.PORT;
const morgan = require('morgan');
const os = require('os');
app.use(morgan('dev'));

app.get('/details', (req, res) => {
  res.send({
    message: 'This is the details endpoint!',
    nodeVersion: process.version,
    hostname: os.hostname(),
    platform: process.platform,
    requestMethod: req.method,
    requestUrl: req.originalUrl,
    timestamp: new Date().toISOString()
  });
});

app.get('/healthz', (req, res) => {
  res.status(200).send('OK');
});

app.get('/ready', (req, res) => {
  const isReady = true;
  if (isReady) {
    res.status(200).send('Ready');
  } else {
    res.status(503).send('Not Ready');
  }
});

app.listen(port, () => {
  console.log(`App listening at http://localhost:${port}`);
});
