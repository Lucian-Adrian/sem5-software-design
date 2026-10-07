# System design through one Domi join

System design asks how collaborating parts preserve useful behavior when calls overlap, messages arrive late and processes stop. Use the same group as [OOP foundations](oop-foundations.md): a flat costs 500 per person, has one place, and initially has no members. Alex has budget 550; Jo has 600. Both can afford it. Only one new member may occupy that place.

Required behavior comes first: reject an unaffordable join, never exceed capacity, keep a repeated membership from adding another member, and explain a rejected request. A proposed durable version must also recover committed membership after restart and let a client safely retry an unanswered request. Notifications, if required, need a separate delivery contract.

The current Go labs demonstrate domain rules, a process-local memory store and synchronous observers. The database design below is proposed study material. Its [optional Python/SQLite model](system-design-practice.py) demonstrates a few mechanisms without changing the Go submission language or product. Authentication, persistent storage and durable delivery are not implemented in the labs.

## A displayed place is an observation

A client is the caller, such as a phone app. An API defines the requests it can send and the responses it can receive. Alex's client displays an available place because a previous read saw zero members. Jo's client can display the same thing. That read does not reserve capacity.

An interleaving is the order in which steps from different calls happen:

| Step | Alex's call | Jo's call |
| --- | --- | --- |
| 1 | Reads count 0, capacity 1 | |
| 2 | | Reads count 0, capacity 1 |
| 3 | Decides one place is free | |
| 4 | | Decides one place is free |
| 5 | Adds Alex | |
| 6 | | Adds Jo using the earlier decision |

The final count is two if both additions are retained. A lost update can instead overwrite one member; neither outcome is a correct join protocol. Making each assignment atomic is insufficient because the decision depended on an earlier read. The protected unit must include reading the relevant state, validating it and recording the change.

In [Lab 0 `MemoryStore.JoinGroup`](lab-00/store.go), a mutex gives one caller at a time access to this whole sequence. The next caller sees the changed count. A mutex is a lock shared by cooperating threads in that process. Two independently running servers have different mutexes and, here, different maps. Adding another server would not combine their state or locks. See the existing [mutex explanation](lab-00/learn.md#why-memorystore-uses-a-mutex).

## Boundaries and authority

The client collects intent and displays results. The HTTP boundary parses input, limits request size and translates errors. Authentication establishes who called; authorization decides whether that caller may act for the specified profile. The domain defines affordability, duplicate and capacity rules. The store supplies authoritative state and a safe way to change it.

Current [HTTP handlers](lab-00/http.go) accept a profile ID from JSON. That is input, not proof of identity. A proposed authenticated route must bind the profile to the verified caller or explicitly check delegated permission. Looking up a real profile alone is insufficient. Recheck authority on a retry before returning private stored results.

A source of truth is the record whose committed state decides an outcome. In the lab it is this process's `MemoryStore`. In the proposal it is one logical database used by every writer. A database stores structured records; a table holds rows such as groups or memberships. A connection is one caller's session with that database. Separate connections can access the same data, unlike separate Go maps.

```text
CURRENT LAB
client -> HTTP handler -> service -> MemoryStore mutex + maps
                                      |
                                      +-> domain membership rule
Lab 3 dispatcher -> command -> service -> success callbacks in this process

PROPOSED DURABLE VERSION
client -> authenticated API -> join service
                                  |
                                  v
                         one database transaction
                         membership + request result + outbox intent
                                  |
                            later worker -> external effect
```

An arrow names a call or dependency, not a separate machine. Keep one deployable application initially. Neither these boundaries nor their interfaces require microservices.

## Specify the API and retry contract

An HTTP response can disappear after a successful write. A timeout therefore means the caller lacks an answer; it does not prove the join failed. In the current lab, repeating the same profile/group join preserves membership because the domain checks duplicates first. But a retry reads the current group detail, not a saved response, and Lab 3 notifies observers after each successful invocation. These are different guarantees.

HTTP defines idempotence in terms of the intended effect of repeating an operation. POST has no general idempotence guarantee; an application can define a suitable retry protocol. See [RFC 9110, idempotent methods](https://www.rfc-editor.org/rfc/rfc9110.html#name-idempotent-methods).

For a proposed `POST /groups/{id}/join`, the client creates a request key once per intended action and reuses it for retries. Store the key with the authenticated actor, normalized request fields and the result in the same transaction as the join. A matching repeat returns the stored status/body. Reusing that actor/key for different request fields is rejected. The key is not a credential, and a changed key does not bypass membership uniqueness.

| Outcome | Current lab | Proposed retry behavior |
| --- | --- | --- |
| Valid join or existing membership | 200, current group detail | 200; committed request result replayed for the same key |
| Invalid JSON/fields | 400 | Reject before attempting a join |
| Missing group/profile | 404 | Stable error contract; reveal only permitted information |
| Budget insufficient | 422 | Store a completed domain outcome if this is the chosen contract |
| Full group | 409 | Store that outcome; a later new attempt needs a new key |
| Unauthenticated/forbidden | No auth layer | 401/403 according to the authentication/authorization contract |
| Key reused for different intent | No key support | 409 with a distinct error code |
| Database temporarily unavailable | No database | Temporary failure, not a fabricated capacity rejection |

Specify key scope, retention and replay fields. This study proposal retains completed keys throughout the exercise; an expiry policy would bound its replay guarantee. A stored success may contain an older member count, so distinguish the operation's historical result from a fresh group read. Do not cache unexpected infrastructure failures as if the domain had decided. Clients should retry temporary uncertainty with bounded delays and the same key, rather than multiply concurrent attempts.

## Make the decision durable and atomic

A transaction groups database operations that commit together or roll back together. Commit makes the group of changes visible as completed; rollback discards the uncommitted changes. Isolation controls which intermediate states another transaction can observe. Durability means committed state is retained according to the database's storage guarantees, rather than depending on a surviving application process.

The proposed transaction first checks for a matching request record. Otherwise it reads the authoritative group/profile, recognizes existing membership, checks budget and capacity, inserts a new membership when eligible, records any required event intent, and stores the response. Then it commits. Return success after commit. An unexpected failure rolls the transaction back.

A unique constraint prevents duplicate values in a specified key, such as `(group_id, profile_id)` or `(actor, request_key)`. It is useful protection but does not enforce capacity across several membership rows. Capacity also requires a shared serialization mechanism: for example, a locked group row with all join writers following the protocol, or a conditional capacity update in the same transaction as membership insertion. Merely writing `BEGIN` around unrelated reads and writes is not a universal race solution; the database's isolation and locking rules matter.

If concurrent transactions both initially find no request key, the unique key decides which record can commit. On a key conflict, roll back the losing transaction, start a fresh transaction, read the committed record, compare the full normalized intent, then replay or reject. Never replay solely because a key exists. SQLite serializes the toy's writers globally, so its changed-intent check does not exercise this race.

The optional model uses SQLite `BEGIN IMMEDIATE`, which acquires its write transaction before reading membership. SQLite permits one writer at a time, so another connection's write attempt can report busy. After the first commits, the second starts a fresh transaction and observes the full group. The model serializes writes for the database, not only the selected group. This teaches correctness at the cost of write concurrency. See [SQLite transaction control](https://www.sqlite.org/lang_transaction.html).

Durable storage is still a system dependency. Hardware, database configuration and recovery procedures affect failure guarantees. A committed transaction is not a claim of surviving every storage failure, and copying a live database file is not automatically a consistent backup. Those operational questions follow after the local transaction is understood.

## Walk through interruption and replay

Start with zero members and no request result. Alex sends key `a`:

1. If the process stops before commit, the uncommitted membership, response and event intent must not become a completed operation. A retry can attempt the action again.
2. If commit succeeds but the HTTP response is lost, the database already holds Alex, key `a` and its success result. The retry finds key `a`, returns that result and inserts nothing.
3. If Jo then tries another key, a fresh transaction sees the full group and records a rejection. Replaying Alex's key still returns Alex's committed success.

The client cannot determine which branch occurred from a timeout alone. Recording the result beside the state change removes the need to guess. The teaching program explicitly rolls back a staged transaction and reopens the database after closing connections. Those observations demonstrate rollback and committed replay; they are not a power-loss or process-kill experiment.

## An event intent is not a delivered notification

A queue holds work for later processing. It helps when an external action is slow, unavailable, or must be retried independently. It adds state and failure handling, so do not add one merely to execute the fast local join later.

Sending a message and updating a database are two separate effects. Sending first can announce a join that later rolls back. Committing first and then sending can lose the notification if the process stops between them. An outbox stores the intent to publish in the same database transaction as membership. A worker later reads committed intents and attempts the external action. This closes the missing-intent gap; it does not make the external action atomic with the transaction. See [transactional outbox guidance](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html).

Suppose a worker sends successfully, then stops before marking the outbox row completed. It must try again after recovery. That is why retries can deliver more than once. At-least-once processing aims to keep retrying accepted work until it is handled, under assumptions about eventual recovery and successful processing. It does not mean a permanently invalid message inevitably succeeds. The [RabbitMQ reliability guide](https://www.rabbitmq.com/docs/reliability#data-safety-on-the-consumer-side) explains redelivery and consumer idempotence.

Give events stable IDs; consumers can recognize repeated IDs. For a consumer that changes its own database, storing the processed ID and its effect in one transaction can suppress repeated effects there. Email involves another system: a local “processed” flag alone cannot atomically cover an external send. Provider-supported idempotency or an explicit duplicate policy may help, but this material implements neither. Worker claiming, retry delays and failed-work handling would also need a contract before deployment.

The actual [Lab 3 dispatcher](lab-03/behavioral.go) immediately invokes callbacks after a command succeeds. They are not durable, and a panic interrupts later callbacks without undoing membership. It reports successful invocation, including duplicate joins. The toy outbox instead records `MemberJoined` only for a new membership. Different event meanings must be stated before comparing counts; see [the existing observer defense](lab-03/learn.md#notify-listeners-after-success).

## Read speed, capacity and availability

A cache is a derived copy used to answer reads faster. Its data can lag behind the source of truth. A short expiration limits how long a cached value is reused; it does not reserve a displayed place or prevent a write race. Serve a listing from a cache only if the resulting stale display is acceptable. Decide the join against authoritative transactional state. Start without a cache when measured read load does not justify its extra rules.

Estimate load with explicit assumptions. Suppose a proposed pilot has 100 active users, each viewing a list once per ten seconds: about 10 listing requests per second. If one in twenty views produces a join, that is about 0.5 joins per second. These are illustrative inputs, not measured Domi traffic. A tenfold peak becomes 100 reads and five joins per second; a burst targeting one flat still competes for one place.

Measure latency, throughput and contention before adding servers. Latency is time per operation; throughput is operations completed per unit time. More API servers can help independent request work if they share the authoritative store, but they cannot create extra places or remove a hot group's serialization requirement. More servers around the lab's separate memory maps would instead create inconsistent answers.

Availability asks whether useful requests can receive acceptable responses during failures. If the authoritative database is unreachable, this design rejects or defers a join rather than promising a place from stale state. Listings might still be shown with appropriate freshness expectations. A second application instance does not eliminate a single database dependency. Database replication and failover add their own freshness and recovery questions; introduce them when an explicit downtime requirement warrants the cost.

Observability supplies evidence for diagnosis: request correlation IDs, operation duration, domain outcome counts, database busy/error counts, and oldest pending outbox age. A capacity rejection is different from an infrastructure failure. Avoid putting private budgets or authentication tokens into logs. A quick HTTP success metric also says nothing about a notification still waiting in the outbox. Start with measurements that distinguish these concrete failures.

## Run the optional model

Read `SCHEMA`, `join_in_transaction`, `join`, then `demonstrate` in [system-design-practice.py](system-design-practice.py). Tables hold state; `SELECT` reads it, `INSERT` adds a row, and `?` placeholders pass data separately from SQL text. `PRIMARY KEY` supplies uniqueness. The actor argument stands in for an already-authenticated identity. The connection's explicit transaction is the caller's responsibility; the helper requires it.

```sh
cd /d/edu/faf/sem5/software-design
python3 system-design-practice.py
```

The standard-library model uses two independent connections with a deliberate interleaving, not two HTTP servers. Connection one holds a write transaction; connection two really encounters the database lock, then retries after commit. Assertions check one joined member, the other caller's full result, identical request replay, a new key for existing membership, changed-intent rejection, rollback and reopening. It creates and removes a temporary database. Expected output:

```text
second connection: busy while first holds write transaction
final seat: alex=joined jo=full
same request replay: outcome=joined members=1 replayed=True
new key same member: outcome=already_member members=1
committed rows: members=1 requests=3 outbox=1
same key changed group: rejected
rollback before commit: no membership/request/outbox survives
reopened database: committed request replays; one outbox intent
temporary database removed; no worker or external effect executed
```

The three request rows belong to Alex's first key, Jo's rejection and Alex's new key. Only Alex's first request inserts membership and an outbox intent. The simplified schema is not a production integrity/security design; it assumes its writers use this transaction protocol. It has no HTTP/authentication implementation, worker, provider, distributed-host test or persistent deployment.

## Reconstruct and change the conditions

Study in three proposed sessions after the OOP route: 50 minutes for boundaries/interleaving, 60 for transactions/retry and the model, and 50 for effects/cache/load. Include reading, reconstruction and correction within each session. Later retrieval needs additional time. These are study suggestions, not calendar appointments or a mastery estimate.

Answer before consulting the checks:

1. Draw two calls competing for the last place. Mark the first stale observation and the exact interval that must be protected. Change the case to two application servers with different mutexes.
2. Reconstruct the transaction in pseudocode, including request lookup, duplicate membership, capacity and commit. A success response is lost. What must the retry read? Then reuse that key for another group.
3. Draw membership, response and outbox rows before commit, after rollback and after commit. Explain why a unique membership key alone does not enforce capacity.
4. A worker crashes after sending but before recording completion. Design the next attempt and state the unresolved external-effect guarantee. Change the requirement: no notification is needed at all.
5. A cache reports one free place while the database is full. Where should the join decide? Change the load assumptions to 200 users viewing once per five seconds, with one join per twenty views. What rates follow, and what still needs measurement?

## Checking answers

1. Both reads can see zero. Protect authoritative read/check/write as one coordinated operation. Two local mutexes protect different processes; use a shared database protocol for the same capacity invariant.
2. Look up actor/key and compare intent before doing new work. Replay the committed result after lost response; reject changed intent. A fresh key still recognizes existing membership and must not produce another insertion event.
3. All three changes belong to one transaction. Rollback retains none; commit retains all. Uniqueness prevents Alex appearing twice, but different profiles can each be unique while collectively exceeding one place.
4. Retry may resend. A stable event ID supports deduplication where the effect and processed-ID record can be coordinated. It does not itself prove exactly-once email. If no downstream action is required, remove the outbox/worker from the proposed design.
5. The transactional database decides. The changed assumptions give 40 reads and two joins per second on average. Measure peak distribution, query cost, lock waits and response time before choosing caching or more servers. Average rates alone do not characterize bursts or a contested group.

An independent explanation, a working reconstruction and a fresh changed case can reveal the next learning gap. Producing this guide and running its model do not establish that you can yet design or defend the system independently.
