"""Optional SQLite teaching model. Run: python3 system-design-practice.py.

Two connections model competing callers with an explicit, deterministic
interleaving. This is not an HTTP service, distributed deployment or worker.
"""

import sqlite3
import tempfile
from contextlib import closing
from pathlib import Path


SCHEMA = """
CREATE TABLE groups (id TEXT PRIMARY KEY, cost INTEGER, capacity INTEGER);
CREATE TABLE profiles (id TEXT PRIMARY KEY, budget INTEGER);
CREATE TABLE members (
    group_id TEXT, profile_id TEXT,
    PRIMARY KEY (group_id, profile_id)
);
CREATE TABLE requests (
    actor TEXT, request_key TEXT, group_id TEXT, outcome TEXT, member_count INTEGER,
    PRIMARY KEY (actor, request_key)
);
CREATE TABLE outbox (
    id INTEGER PRIMARY KEY, kind TEXT, group_id TEXT, profile_id TEXT
);
INSERT INTO groups VALUES ('flat', 500, 1), ('other', 500, 1);
INSERT INTO profiles VALUES ('alex', 550), ('jo', 600), ('pat', 400);
"""


def join_in_transaction(connection, actor, request_key, group_id):
    """Caller holds a write transaction and has already authenticated actor."""
    previous = connection.execute(
        "SELECT group_id, outcome, member_count FROM requests "
        "WHERE actor=? AND request_key=?", (actor, request_key)
    ).fetchone()
    if previous is not None:
        if previous[0] != group_id:
            raise ValueError("request key reused for a different group")
        return previous[1], previous[2], True

    group = connection.execute(
        "SELECT cost, capacity FROM groups WHERE id=?", (group_id,)
    ).fetchone()
    profile = connection.execute(
        "SELECT budget FROM profiles WHERE id=?", (actor,)
    ).fetchone()
    if group is None or profile is None:
        raise ValueError("unknown group or profile")

    count = connection.execute(
        "SELECT COUNT(*) FROM members WHERE group_id=?", (group_id,)
    ).fetchone()[0]
    member = connection.execute(
        "SELECT 1 FROM members WHERE group_id=? AND profile_id=?", (group_id, actor)
    ).fetchone()
    if member is not None:
        outcome = "already_member"
    elif profile[0] < group[0]:
        outcome = "budget_too_low"
    elif count >= group[1]:
        outcome = "full"
    else:
        connection.execute("INSERT INTO members VALUES (?, ?)", (group_id, actor))
        connection.execute(
            "INSERT INTO outbox(kind, group_id, profile_id) VALUES ('MemberJoined', ?, ?)",
            (group_id, actor)
        )
        count += 1
        outcome = "joined"

    connection.execute("INSERT INTO requests VALUES (?, ?, ?, ?, ?)",
                       (actor, request_key, group_id, outcome, count))
    return outcome, count, False


def join(connection, actor, request_key, group_id):
    connection.execute("BEGIN IMMEDIATE")
    try:
        result = join_in_transaction(connection, actor, request_key, group_id)
        connection.execute("COMMIT")
        return result
    except Exception:
        connection.execute("ROLLBACK")
        raise


def counts(connection):
    return tuple(connection.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
                 for table in ("members", "requests", "outbox"))


def demonstrate(path):
    # Explicit SQL controls transactions; zero timeout makes contention observable.
    with closing(sqlite3.connect(path, isolation_level=None, timeout=0)) as first:
        with closing(sqlite3.connect(path, isolation_level=None, timeout=0)) as second:
            first.executescript(SCHEMA)
            first.execute("BEGIN IMMEDIATE")
            try:
                second.execute("BEGIN IMMEDIATE")
            except sqlite3.OperationalError as error:
                assert "locked" in str(error), error
                print("second connection: busy while first holds write transaction")
            else:
                second.execute("ROLLBACK")
                raise AssertionError("expected independent connection contention")

            alex = join_in_transaction(first, "alex", "request-a", "flat")
            first.execute("COMMIT")
            jo = join(second, "jo", "request-j", "flat")
            replay = join(second, "alex", "request-a", "flat")
            repeat = join(second, "alex", "new-key", "flat")
            assert alex == ("joined", 1, False)
            assert jo == ("full", 1, False)
            assert replay == ("joined", 1, True)
            assert repeat == ("already_member", 1, False)
            print("final seat: alex=joined jo=full")
            print("same request replay: outcome=joined members=1 replayed=True")
            print("new key same member: outcome=already_member members=1")
            assert counts(second) == (1, 3, 1)
            print("committed rows: members=1 requests=3 outbox=1")

            try:
                join(second, "alex", "request-a", "other")
            except ValueError:
                print("same key changed group: rejected")
            else:
                raise AssertionError("expected request fingerprint mismatch")
            assert counts(second) == (1, 3, 1)

            # Rollback models interruption before commit, not a power-loss experiment.
            first.execute("BEGIN IMMEDIATE")
            assert join_in_transaction(first, "jo", "request-other", "other")[0] == "joined"
            first.execute("ROLLBACK")
            assert counts(second) == (1, 3, 1)
            print("rollback before commit: no membership/request/outbox survives")

    # A fresh connection can read a committed result after the earlier ones close.
    with closing(sqlite3.connect(path, isolation_level=None)) as reopened:
        assert join(reopened, "alex", "request-a", "flat") == ("joined", 1, True)
        assert counts(reopened) == (1, 3, 1)
        print("reopened database: committed request replays; one outbox intent")


def main():
    with tempfile.TemporaryDirectory(prefix="tmps-system-design-") as directory:
        demonstrate(Path(directory) / "practice.sqlite")
    print("temporary database removed; no worker or external effect executed")


if __name__ == "__main__":
    main()
