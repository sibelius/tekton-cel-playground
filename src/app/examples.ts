export type Example = {
  name: string;
  description: string;
  celExpression: string;
  httpRequest: string;
};

export const EXAMPLES: Example[] = [
  {
    name: "GitHub push to path",
    description:
      "Match a GitHub push event that touches files under service-pix-spi/.",
    celExpression: `header.match('x-github-event', 'push') &&
body.commits.exists(commit,
  commit.modified.exists(p, p.startsWith('service-pix-spi/')) ||
  commit.added.exists(p, p.startsWith('service-pix-spi/')) ||
  commit.removed.exists(p, p.startsWith('service-pix-spi/'))
)`,
    httpRequest: `POST /webhook HTTP/1.1
Host: ci.example.com
Content-Type: application/json
X-GitHub-Event: push

{
  "ref": "refs/heads/main",
  "commits": [
    {
      "id": "abc123",
      "modified": ["service-pix-spi/handler.go"],
      "added": [],
      "removed": []
    },
    {
      "id": "def456",
      "modified": ["docs/readme.md"],
      "added": [],
      "removed": []
    }
  ]
}`,
  },
  {
    name: "GitHub pull_request opened",
    description: "Trigger only when a PR is opened against main.",
    celExpression: `header.match('x-github-event', 'pull_request') &&
body.action == 'opened' &&
body.pull_request.base.ref == 'main'`,
    httpRequest: `POST /webhook HTTP/1.1
Host: ci.example.com
Content-Type: application/json
X-GitHub-Event: pull_request

{
  "action": "opened",
  "pull_request": {
    "number": 42,
    "base": { "ref": "main" },
    "head": { "ref": "feature/cel" }
  }
}`,
  },
  {
    name: "Header canonical()",
    description:
      "Read a single header value (first occurrence, case-insensitive).",
    celExpression: `header.canonical('content-type') == 'application/json'`,
    httpRequest: `POST /api HTTP/1.1
Host: example.com
Content-Type: application/json

{"hello": "world"}`,
  },
  {
    name: "Body field equality",
    description: "Combine a header check with a body field comparison.",
    celExpression: `header.match('x-event', 'order.created') &&
body.order.total > 100 &&
body.order.currency == 'BRL'`,
    httpRequest: `POST /events HTTP/1.1
Host: api.example.com
Content-Type: application/json
X-Event: order.created

{
  "order": {
    "id": "ord_123",
    "total": 250,
    "currency": "BRL"
  }
}`,
  },
  {
    name: "Method / URL filter",
    description:
      "Filter by HTTP method and request path — useful for routing logic.",
    celExpression: `method == 'POST' && requestURL.startsWith('/webhook/')`,
    httpRequest: `POST /webhook/github HTTP/1.1
Host: example.com
Content-Type: application/json

{"ok": true}`,
  },
  {
    name: "Tag push only",
    description: "Match push events that target a tag ref.",
    celExpression: `header.match('x-github-event', 'push') &&
body.ref.startsWith('refs/tags/')`,
    httpRequest: `POST /webhook HTTP/1.1
Host: ci.example.com
Content-Type: application/json
X-GitHub-Event: push

{
  "ref": "refs/tags/v1.2.3",
  "commits": []
}`,
  },
];
