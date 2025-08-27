import { DocEntry } from '../types.js';
import { hybridSearch } from '../retr/search.js';

export function createSearchHandler(dataDir: string, topN: number) {
  return async (query: string): Promise<DocEntry[]> => {
    try {
      const results = await hybridSearch(dataDir, query, topN);
      return results.map(result => result.metadata);
    } catch (error) {
      return [];
    }
  };
}

export function createExpressSearchHandler(dataDir: string, topN: number) {
  return async (req: any, res: any) => {
    try {
      const { query } = req.body;
      
      if (!query || typeof query !== 'string') {
        return res.status(400).json({ error: 'Query parameter is required' });
      }

      const results = await hybridSearch(dataDir, query, topN);
      const docEntries = results.map(result => result.metadata);
      
      res.json(docEntries);
    } catch (error) {
      console.error('Search error:', error);
      res.status(500).json({ error: 'Internal server error' });
    }
  };
}