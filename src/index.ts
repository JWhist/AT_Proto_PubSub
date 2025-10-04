import { Firehose, CommitEvt, Event } from '@atproto/sync';
import { IdResolver } from '@atproto/identity';

interface FilterOptions {
  repository?: string;
  keyword?: string;
}

class FirehoseFilterServer {
  private firehose: Firehose;
  private filters: FilterOptions;
  private idResolver: IdResolver;

  constructor(filters: FilterOptions = {}) {
    this.filters = filters;
    this.idResolver = new IdResolver();
    
    this.firehose = new Firehose({
      idResolver: this.idResolver,
      service: 'wss://bsky.network',
      handleEvent: this.handleEvent.bind(this),
      onError: this.handleError.bind(this),
    });
  }

  private async handleEvent(evt: Event): Promise<void> {
    try {
      // We're mainly interested in commit events (create, update, delete)
      if (evt.event === 'create' || evt.event === 'update') {
        const commitEvt = evt as CommitEvt;
        
        // Filter by repository if specified
        if (this.filters.repository && commitEvt.did !== this.filters.repository) {
          return;
        }

        // Check if the record matches our filter
        if (this.matchesFilter(commitEvt)) {
          this.logEvent(commitEvt);
        }
      }
    } catch (error) {
      console.error('Error handling event:', error);
    }
  }

  private handleError(err: Error): void {
    // Log error but continue running - connection will auto-retry
    console.error('Firehose error:', err.message);
    if (err.message.includes('getaddrinfo') || err.message.includes('ENOTFOUND')) {
      console.error('Network connectivity issue - check your internet connection');
    }
  }

  private matchesFilter(evt: CommitEvt): boolean {
    if (evt.event !== 'create' && evt.event !== 'update') {
      return false;
    }

    const record = evt.record;
    
    // Check if record has text content
    if (!record || typeof record !== 'object') {
      return false;
    }

    // Look for text in common fields
    const text = (record as any).text || (record as any).message || (record as any).content || '';

    // If no keyword filter, match all records with text
    if (!this.filters.keyword) {
      return !!text;
    }

    // Check if text contains keyword (case-insensitive)
    if (typeof text === 'string') {
      return text.toLowerCase().includes(this.filters.keyword.toLowerCase());
    }

    return false;
  }

  private logEvent(evt: CommitEvt) {
    const timestamp = new Date().toISOString();
    console.log('='.repeat(80));
    console.log(`[${timestamp}] ${evt.event.toUpperCase()} event`);
    console.log('-'.repeat(80));
    console.log(`Repository: ${evt.did}`);
    console.log(`Collection: ${evt.collection}`);
    console.log(`Record Key: ${evt.rkey}`);
    console.log(`URI: ${evt.uri.toString()}`);
    
    if (evt.event === 'create' || evt.event === 'update') {
      const record = evt.record as any;
      
      // Log text content
      const text = record.text || record.message || record.content || '';
      if (text) {
        console.log(`Text: ${text}`);
      }

      // Log other relevant fields
      if (record.reply) {
        console.log(`Reply to: ${JSON.stringify(record.reply)}`);
      }
      
      if (record.langs) {
        console.log(`Languages: ${JSON.stringify(record.langs)}`);
      }
    }

    console.log('='.repeat(80));
    console.log();
  }

  public start() {
    console.log('Starting AT Protocol Firehose Filter Server...');
    console.log('Filters:');
    console.log(`  Repository: ${this.filters.repository || 'ALL'}`);
    console.log(`  Keyword: ${this.filters.keyword || 'ALL'}`);
    console.log('Connecting to firehose...\n');

    this.firehose.start();
  }

  public async stop() {
    console.log('Stopping firehose...');
    await this.firehose.destroy();
  }
}

// Parse command line arguments
function parseArgs(): FilterOptions {
  const args = process.argv.slice(2);
  const filters: FilterOptions = {};

  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--repository' || args[i] === '-r') {
      filters.repository = args[++i];
    } else if (args[i] === '--keyword' || args[i] === '-k') {
      filters.keyword = args[++i];
    } else if (args[i] === '--help' || args[i] === '-h') {
      console.log('AT Protocol Firehose Filter Server');
      console.log('\nUsage: npm start -- [options]');
      console.log('   or: node dist/index.js [options]');
      console.log('\nOptions:');
      console.log('  -r, --repository <repo>  Filter by repository DID');
      console.log('  -k, --keyword <keyword>  Filter by keyword in text');
      console.log('  -h, --help               Show this help message');
      console.log('\nExample:');
      console.log('  npm start -- --keyword "hello"');
      console.log('  node dist/index.js --repository did:plc:abc123 --keyword "test"');
      process.exit(0);
    }
  }

  return filters;
}

// Main execution
const filters = parseArgs();
const server = new FirehoseFilterServer(filters);

// Handle graceful shutdown
process.on('SIGINT', async () => {
  console.log('\nReceived SIGINT, shutting down...');
  await server.stop();
  process.exit(0);
});

process.on('SIGTERM', async () => {
  console.log('\nReceived SIGTERM, shutting down...');
  await server.stop();
  process.exit(0);
});

server.start();
