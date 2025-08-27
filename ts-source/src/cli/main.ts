#!/usr/bin/env node

import { Command } from 'commander';
import { readFile } from 'fs/promises';
import { fetchArtifact } from '../internal/fetch/index.js';
import { prepareDirs, processArtifact } from '../internal/pipe/index.js';
import { buildIndex } from '../internal/index/index.js';
import { hybridSearch, formatSearchResults } from '../internal/retr/index.js';
import { createExpressSearchHandler } from '../internal/api/index.js';

const program = new Command();

program
  .name('kb')
  .description('A minimal proof-of-concept knowledge base tool for document ingestion, indexing, and search')
  .version('1.0.0');

program
  .command('ingest')
  .description('Ingest documents from a list file')
  .requiredOption('-l, --list <file>', 'path to a file containing newline separated URLs or file paths')
  .option('-d, --data <dir>', 'directory in which to store artefacts and index files', './data')
  .option('-t, --github-token <token>', 'GitHub personal access token for private repositories', '')
  .action(async (options) => {
    try {
      await prepareDirs(options.data);

      const listContent = await readFile(options.list, 'utf-8');
      const lines = listContent.split('\n').map(line => line.trim()).filter(line => line && !line.startsWith('#'));

      for (const line of lines) {
        try {
          console.log(`Processing: ${line}`);
          const artifact = await fetchArtifact(line, options.githubToken);
          await processArtifact(artifact, options.data);
        } catch (error) {
          console.error(`Skipping ${line}: ${error instanceof Error ? error.message : String(error)}`);
        }
      }

      await buildIndex(options.data);
      console.log('Ingestion complete');
    } catch (error) {
      console.error(`Error: ${error instanceof Error ? error.message : String(error)}`);
      process.exit(1);
    }
  });

program
  .command('search')
  .description('Search the knowledge base')
  .option('-d, --data <dir>', 'directory containing the index', './data')
  .option('-t, --top <n>', 'number of top results to display', '10')
  .argument('<query>', 'search query')
  .action(async (query, options) => {
    try {
      const topN = parseInt(options.top, 10);
      const results = await hybridSearch(options.data, query, topN);

      if (results.length === 0) {
        console.log('No matches found');
        return;
      }

      console.log(formatSearchResults(results));
    } catch (error) {
      console.error(`Search error: ${error instanceof Error ? error.message : String(error)}`);
      process.exit(1);
    }
  });

program
  .command('serve')
  .description('Start HTTP server for search API')
  .option('-d, --data <dir>', 'directory containing the index', './data')
  .option('-a, --addr <address>', 'address to listen on', ':8080')
  .option('-t, --top <n>', 'number of results to return', '10')
  .action(async (options) => {
    console.log(`HTTP server functionality is stubbed out in this implementation.`);
    console.log(`In a full implementation, this would:`);
    console.log(`1. Start an Express server on ${options.addr}`);
    console.log(`2. Serve POST /search endpoint`);
    console.log(`3. Return top ${options.top} results from data directory: ${options.data}`);
    console.log(``);
    console.log(`To implement this, you would use Express.js:`);
    console.log(``);
    console.log(`import express from 'express';`);
    console.log(`const app = express();`);
    console.log(`app.use(express.json());`);
    console.log(`app.post('/search', createExpressSearchHandler('${options.data}', ${options.top}));`);
    console.log(`app.listen(port, () => console.log('Server running on port', port));`);
  });

program.parse();