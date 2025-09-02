require('dotenv').config();
const express = require('express');
const app = express();
const port = process.env.PORT;

app.get('/details', (req, res) => {
  res.send('Hello World!');
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
