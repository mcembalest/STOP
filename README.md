# STOP

One **GOAL**. As many **ROLEs** as the goal needs.

Write a file like [example.stop.md](example.stop.md):

```md
# GOAL

Decide whether to start a reading group about AI agents.

# ROLE Organizer

Suggest a small format I could host next month.

# ROLE Skeptic

Find the reasons it might fail and a cheap way to test interest.
```

That is the whole language. A role states a responsibility. STOP sets no sequence or loop.

Run it:

```sh
STOP_MODEL=gpt-5.5 go run . example.stop.md
```

The small Go runner asks each role to work on the goal and prints their answers. Redirect the output to a file if you want to keep it. `STOP_MODEL` chooses a model supported by your [Codex CLI](https://developers.openai.com/codex/cli); omit it to use your CLI default.

To use another agent for a role, pass an executable that reads a prompt on stdin and writes an answer on stdout:

```sh
go run . -agent Skeptic=/absolute/path/to/my-agent example.stop.md
```

The executable receives `STOP_ROLE` and `STOP_GOAL` as environment variables. Agent choice is outside the STOP file.
