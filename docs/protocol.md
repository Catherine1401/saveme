# Socket protocol

The production socket is `/run/saveme.sock`, mode `0600`, owned by the configured ordinary account. The daemon additionally checks Linux `SO_PEERCRED`; file ownership alone does not authorize a connection. Each request and response is one JSON line. Requests have a maximum line size of 64 KiB, reject unknown fields, and cannot contain a second JSON value.

Use one connection for a pending operation and its confirmation. There is one pending request globally; a new valid preparation replaces the previous request. Closing the owning connection cancels its pending request. Other connections cannot confirm or cancel it.

| Command | Fields | Result |
| --- | --- | --- |
| `status` | none | Current status; state omitted while active |
| `add_domain` | `domain` | Normalize and add a unique domain |
| `edit_domain` | `index`, `domain` | Normalize and replace a domain |
| `delete_domain` | `index` | Delete a domain; preserve the last one if a lock exists |
| `create_daily` | `windows: [{start, end}]` | Prepare a nonempty daily schedule |
| `add_window` | `start`, `end` | Prepare a new daily window |
| `delete_window` | `index`, optional `confirmed` | Delete a window; the last requires `confirmed: true` |
| `create_range` | `duration` | Prepare one of ten choices: 1h, 2h, 4h, 6h, 12h, 1d, 3d, 7d, 14d, 30d |
| `confirm` | `code` | Recheck guards, persist and enforce the prepared lock |
| `cancel` | none | Cancel the connection's pending request |

Window times use exactly `HH:MM`. Indexes are zero-based and must be present for indexed commands. While active, every command except `status` receives `ERR_LOCKED`, including `cancel`. With no active lock, an untrusted clock receives `ERR_CLOCK_UNSYNC` for mutations.

A preparation response includes `ok`, `pending`, `code`, `expires_in`, `failures`, `now`, and `preview`. `preview` contains the normalized candidate state for the confirmation screen. A wrong code returns `ERR_BAD_CODE` and, until the third failure, a replacement code with the original expiry deadline. A new preparation gets its own 60-second lifetime. Range start/end times in previews are estimates; confirmation recalculates them from the effective confirmation time.

Status responses include `ok`, `active`, and `now`. An existing lock adds `kind` and its next transition as `end_at` if active or `next_at` if inactive. A continuously active daily schedule, or a lock held due to untrusted time, omits `end_at`. Inactive status includes `state`; active status never includes `state`, `preview`, domains, or windows.

Errors from the specification are `ERR_LOCKED`, `ERR_OVERLAP`, `ERR_INVALID`, `ERR_HAS_LOCK`, `ERR_CLOCK_UNSYNC`, `ERR_BAD_CODE`, and `ERR_CODE_EXPIRED`. Storage and network failures additionally use `ERR_STORAGE` and `ERR_ENFORCEMENT`. Unauthorized peers are disconnected before requests are processed. On `ERR_STORAGE` or `ERR_ENFORCEMENT`, inspect `status` and service logs: a failed response does not guarantee that a partially persisted operation has been rolled back.

The state envelope contains a generation, state payload, and SHA-256 checksum. State records are limited to 1 MiB. A load selects the newest valid generation from the two files. Pending codes are never persisted.
