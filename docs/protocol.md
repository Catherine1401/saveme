# Socket protocol

The production socket is `/run/saveme.sock`, mode `0600`, owned by the configured ordinary account. The daemon additionally checks Linux `SO_PEERCRED`; file ownership alone does not authorize a connection. Each request and response is one JSON line. Requests have a maximum line size of 64 KiB, reject unknown fields, and cannot contain a second JSON value.

Use one connection for a pending operation and its confirmation. There is one pending request globally; a new valid preparation replaces the previous request. Closing the owning connection cancels its pending request. Other connections cannot confirm or cancel it.

| Command | Fields | Result |
| --- | --- | --- |
| `status` | none | Current status with one `entries` item per domain |
| `add_domain` | `domain` | Normalize and add a unique, unlocked domain |
| `edit_domain` | `index`, `domain` | Normalize and replace an unlocked domain |
| `delete_domain` | `index`, optional `confirmed` | Delete an unlocked domain; a domain with a stored schedule requires `confirmed: true` and loses the schedule |
| `create_daily` | `entry`, `windows: [{start, end}]` | Prepare a nonempty daily schedule for one domain |
| `add_window` | `entry`, `start`, `end` | Prepare a new daily window for one domain |
| `edit_window` | `entry`, `index`, `start`, `end` | Prepare a replacement daily window; confirmation required |
| `delete_window` | `entry`, `index`, optional `confirmed` | Delete a window; the last requires `confirmed: true` and removes the schedule |
| `create_range` | `entry`, `duration` | Prepare one of ten choices for one domain: 1h, 2h, 4h, 6h, 12h, 1d, 3d, 7d, 14d, 30d |
| `confirm` | `code` | Recheck the target domain, persist and enforce the prepared lock |
| `cancel` | none | Cancel the connection's pending request |

Window times use exactly `HH:MM`. Indexes are zero-based and must be present for indexed commands. `index` selects a domain for domain commands and a window for window commands; `entry` selects the domain for schedule commands. Each domain has at most one lock, daily or timed; locks are independent. A command whose target domain is active receives `ERR_LOCKED`; for `confirm`, the target is the pending request's domain. Commands on other domains, `add_domain` and `cancel` are not blocked by another domain's lock. An untrusted clock keeps every stored lock active and returns `ERR_CLOCK_UNSYNC` for every mutation except `add_domain`; a successful `add_domain` then returns the new status with `ok: false` and `ERR_CLOCK_UNSYNC`. Creating a lock for a domain that already has one returns `ERR_HAS_LOCK`.

A preparation response includes `ok`, `pending`, `code`, `expires_in`, `failures`, `now`, and `preview`. `preview` contains the target domain and its candidate lock for the confirmation screen. A wrong code returns `ERR_BAD_CODE` and, until the third failure, a replacement code with the original expiry deadline. A new preparation gets its own 60-second lifetime. Range start/end times in previews are estimates; confirmation recalculates them from the effective confirmation time.

Status responses include `ok`, `active` (true when any domain is active), `now`, and `entries`. Each entry has `domain` and `active`. A domain with a stored lock adds `kind` and `zone`, and its next transition as `end_at` if active or `next_at` if inactive. A continuously active daily schedule, or a lock held due to untrusted time, omits `end_at`. Only inactive domains include `windows`.

Errors from the specification are `ERR_LOCKED`, `ERR_OVERLAP`, `ERR_INVALID`, `ERR_HAS_LOCK`, `ERR_CLOCK_UNSYNC`, `ERR_BAD_CODE`, and `ERR_CODE_EXPIRED`. Storage and network failures additionally use `ERR_STORAGE` and `ERR_ENFORCEMENT`. Unauthorized peers are disconnected before requests are processed. On `ERR_STORAGE` or `ERR_ENFORCEMENT`, inspect `status` and service logs: a failed response does not guarantee that a partially persisted operation has been rolled back.

The state envelope contains a generation, state payload, and SHA-256 checksum. State version 2 stores `entries: [{domain, lock}]`. A version 1 state, which had one shared lock, is verified with its original checksum and loaded by copying that lock to every domain; the next write stores version 2. Earlier binaries cannot read version 2. State records are limited to 1 MiB. A load selects the newest valid generation from the two files. Pending codes are never persisted.
