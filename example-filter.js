// Example of how the firehose filter server works
// This demonstrates the filtering logic without connecting to the live firehose

// Example events that might come from the firehose
const exampleEvents = [
  {
    did: 'did:plc:abc123',
    collection: 'app.bsky.feed.post',
    text: 'Hello world! This is a test post.'
  },
  {
    did: 'did:plc:xyz789',
    collection: 'app.bsky.feed.post',
    text: 'Another post without the keyword.'
  },
  {
    did: 'did:plc:abc123',
    collection: 'app.bsky.feed.post',
    text: 'Testing the firehose filter.'
  },
  {
    did: 'did:plc:def456',
    collection: 'app.bsky.feed.like',
    text: 'I love this test!'
  }
];

function filterEvents(events, repositoryFilter, keywordFilter) {
  return events.filter(event => {
    // Filter by repository if specified
    if (repositoryFilter && event.did !== repositoryFilter) {
      return false;
    }

    // Filter by keyword if specified
    if (keywordFilter && event.text) {
      return event.text.toLowerCase().includes(keywordFilter.toLowerCase());
    }

    // If no keyword filter, include all records with text
    return !!event.text;
  });
}

console.log('Example 1: No filters (all events with text)');
console.log(filterEvents(exampleEvents));
console.log('\n');

console.log('Example 2: Filter by keyword "test"');
console.log(filterEvents(exampleEvents, undefined, 'test'));
console.log('\n');

console.log('Example 3: Filter by repository "did:plc:abc123"');
console.log(filterEvents(exampleEvents, 'did:plc:abc123'));
console.log('\n');

console.log('Example 4: Filter by repository "did:plc:abc123" and keyword "test"');
console.log(filterEvents(exampleEvents, 'did:plc:abc123', 'test'));
