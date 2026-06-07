// Web Push subscription registry for bet-with-goodall.
//
// A tiny HTTP API (API Gateway v2 → this Lambda → DynamoDB) that the static
// site calls to opt in/out of match-result push notifications. The builder
// reads the same table to send pushes. No auth: a push subscription is
// origin-bound and only our VAPID private key can deliver to it, so the worst a
// stranger can do is register their own browser to receive our notifications.
//
// Routes (same-origin via CloudFront under /api/*):
//   POST /api/subscribe    body: a PushSubscription JSON ({ endpoint, keys })
//   POST /api/unsubscribe  body: { endpoint }
//
// Uses only the AWS SDK v3 bundled in the Node.js Lambda runtime — no
// node_modules to package.
import { createHash } from 'node:crypto'
import {
  DynamoDBClient,
  PutItemCommand,
  DeleteItemCommand,
} from '@aws-sdk/client-dynamodb'

const ddb = new DynamoDBClient({})
const TABLE = process.env.TABLE_NAME

// The endpoint URL uniquely identifies a subscription; its hash is the table key
// so we never store an unbounded URL as the partition key directly.
const idFor = (endpoint) => createHash('sha256').update(endpoint).digest('hex')

const json = (statusCode, body) => ({
  statusCode,
  headers: { 'content-type': 'application/json' },
  body: JSON.stringify(body),
})

export const handler = async (event) => {
  const route = event.requestContext?.http
  const path = route?.path || ''
  const method = route?.method || ''

  if (method !== 'POST') {
    return json(405, { error: 'method not allowed' })
  }

  let payload
  try {
    payload = JSON.parse(event.body || '{}')
  } catch {
    return json(400, { error: 'invalid JSON body' })
  }

  if (path.endsWith('/subscribe')) {
    return subscribe(payload)
  }
  if (path.endsWith('/unsubscribe')) {
    return unsubscribe(payload)
  }
  return json(404, { error: 'not found' })
}

async function subscribe(sub) {
  const endpoint = sub?.endpoint
  const p256dh = sub?.keys?.p256dh
  const auth = sub?.keys?.auth
  if (!endpoint || !p256dh || !auth) {
    return json(400, { error: 'missing endpoint or keys' })
  }

  const now = new Date().toISOString()
  await ddb.send(
    new PutItemCommand({
      TableName: TABLE,
      Item: {
        id: { S: idFor(endpoint) },
        endpoint: { S: endpoint },
        p256dh: { S: p256dh },
        auth: { S: auth },
        updated_at: { S: now },
      },
    }),
  )
  return json(200, { ok: true })
}

async function unsubscribe(body) {
  const endpoint = body?.endpoint
  if (!endpoint) {
    return json(400, { error: 'missing endpoint' })
  }
  await ddb.send(
    new DeleteItemCommand({
      TableName: TABLE,
      Key: { id: { S: idFor(endpoint) } },
    }),
  )
  return json(200, { ok: true })
}
