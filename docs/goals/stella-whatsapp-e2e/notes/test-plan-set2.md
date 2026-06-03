# Set 2 — WhatsApp E2E (advanced)

Same runner as set 1: bridge `POST /inject` + `[E2E-…]` markers on **vignesh**.

## Run

```bash
./scripts/whatsapp-e2e/run.sh --set2
./scripts/whatsapp-e2e/run.sh --all    # smoke + extended + set2
python3 scripts/whatsapp-e2e/run_tests.py --case N30
```

## Cases (N*)

| ID | Category | What it checks |
|----|----------|----------------|
| N00 | Policy | `dm_policy: open` in config |
| N01 | Policy | Teddy + Vignesh in `allow_admin_from` |
| N02 | Post-update | `patch-whatsapp-send.log` has `gateway:startup` |
| N10 | Policy | Stranger can DM (open allowlist) |
| N20 | Identity | Reply mentions Stella, not “I am Vignesh” |
| N21 | Identity | Teddy recognized as owner |
| N30 | Third-party | `whatsapp:+6590016046` routes to Gaya, not Teddy |
| N31 | Routing | SET2-N31 text to Gaya only, not home channel |
| N40 | Admin | Teddy `/status` works |
| N41 | Admin | Stranger `/reset` does not reset stranger session |
| N50 | Duplicates | One outbound with `UNIQUE-N50-SET2` |
| N60 | Limits | 600+ char message gets a response |
| N61 | Limits | Refuses dumping `.env` secrets in reply |
| N70 | Co-owners | Names Vignesh (and Teddy) as owners |

## Notes

- N00–N02 are **infra** (no LLM).
- N20/N21/N61 use **soft** text checks — occasional LLM variance.
- Full set2 run ~15–25 min (real agent turns).
