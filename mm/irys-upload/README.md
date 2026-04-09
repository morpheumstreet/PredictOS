# irys-upload

Small HTTP service that uploads JSON to [Irys](https://irys.xyz) paid with **Solana**, matching the contract used by the PredictOS terminal (`POST /api/irys-upload`).

The terminal Bun server proxies to this process so the Node/npm `@irys/upload-solana` stack is not required in the terminal package.

## Dependencies

Uploads use the community module [`github.com/donutnomad/solana-web3/irys`](https://pkg.go.dev/github.com/donutnomad/solana-web3/irys) (Solana signer + Irys bundler HTTP). This is **not** an official Irys SDK; review updates before production use.

## Run

From this directory:

```bash
export IRYS_CHAIN_ENVIRONMENT=devnet   # or mainnet
export IRYS_SOLANA_PRIVATE_KEY=        # base58 keypair (see terminal/.env.example)
export IRYS_SOLANA_RPC_URL=https://api.devnet.solana.com   # required for devnet
# optional: PORT (default 8091)
go run ./cmd/irys-upload
```

Endpoints:

- `GET /status` — `{ "configured": true, "environment": "..." }`
- `POST /upload` — same JSON body as the terminal verifiable-analysis payload

## Environment

| Variable | Required | Notes |
|----------|----------|--------|
| `IRYS_CHAIN_ENVIRONMENT` | yes | `mainnet` or `devnet` |
| `IRYS_SOLANA_PRIVATE_KEY` | yes | Base58-encoded Solana keypair |
| `IRYS_SOLANA_RPC_URL` | devnet only | Mainnet defaults to `https://api.mainnet-beta.solana.com/` if unset |
| `PORT` | no | Default `8091` |

Set `IRYS_UPLOAD_SERVICE_URL` in the **terminal** to `http://127.0.0.1:8091` (or your deployed base URL) so `/api/irys-upload` proxies here.
