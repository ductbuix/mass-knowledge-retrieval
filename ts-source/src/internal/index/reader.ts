import { open } from 'fs/promises';

export async function readPreview(path: string, n: number): Promise<string> {
  try {
    const fileHandle = await open(path, 'r');
    const buffer = Buffer.alloc(n);
    
    try {
      const { bytesRead } = await fileHandle.read(buffer, 0, n, 0);
      return buffer.subarray(0, bytesRead).toString('utf-8');
    } finally {
      await fileHandle.close();
    }
  } catch (error) {
    return '';
  }
}