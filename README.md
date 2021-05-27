# ShellFS

A file system where you can use the output of commands as text files.

Useful when making tarballs from files that don't exist.

There are probably other uses too.

### Implementation

Resources to implement:
https://bazil.org/talks/
https://github.com/bazil/zipfs   -- non-seekable?
https://github.com/bazil/zipfs/blob/master/main.go#L210-L226
https://github.com/bazil/bolt-mount

Would not have been possible without Ty's help on freenode.net#go-nuts

### Scope

Features that are in scope for this project
- Reading from commands
- Writing to commands
- Nesting directories
- Passthrough of normal files

Things too far out of the above, or even on the fringes of the above, may get
closed as out of scope. You are willing to fork and alter for yourself. Pull
requests will be accepted if the problems they solve are in scope.

### Contributing

This repository uses the following commit message format:

```
Problem: The problem we need to fix - 50 chars max

Solution: What we decided to do. This one gets wrapped to 72 characters!
Note how that line is full width, as is the Problem statement? Just as
an example.

Optional paragraphs of further explanation. Such as a defense of this
format.

The author of ZeroMQ, Pieter Hintjens, found that this format captured
the most out-of-band (not explained by source code) details that in a
traditional format would just be lost. I know WHAT a commit does, it has
code, but I want to know WHY. If I want to know why a line of code was
added, it does me no good to see the git blame and a description of
what the other code in that commit does. I want to know what problem was
being solved.

Furthermore, this format, in Pieter's experience, helped keep large
international teams on the same page. Their issue tracker would use this
format, and the code itself would be a series of problems being solved.

Imagine all the times you saw a line that added some flag to a function
call. You check the commit that added it, and see "add flag to call."
Why is that flag there? You have no idea. Now, imagine if you saw:
"Problem: User connections drop regularly." Now you know why that flag
is there! This scales to large chunks of code.

Aside: This format is not suitable in all circumstances. For example
some administrative repositories, such as for gitolite, rarely have
problems to be solved, but rather IT support tickets for new users or
the like. In those cases the purpose of the git repo is not to track
source code authorship (which is fixing a series of problems until the
original problem is solved) but administrative (taking actions, not
necessarily while trying to solve problems but keep an organization
running).

However.

For source code authorship, this format works. It has helped keep teams
on budget, on track, and adds a dimension to the git logs that might
otherwise be lost.
```

Before working on code, please open an issue. Use the same format as the commit
message to format your issue. That means all of your exposition on why it's
a problem, or what you think should be done to fix it, is written in the issue.
Ultimately the Problem: statement, the title of the Issue, will be attached to
one or more commits trying to fix that problem.

If you follow the above process, where you open issues first, flesh out
a problem statement, and then open a PR, your code WILL be merged. If you don't
want your code merged, and want your PR to be a draft, please use `[NOMERGE]`
in the title of your PR.

If you write spam PRs on valid issues, your spammy PRs will become a matter of
git record. Do not test the limits of this, sometimes PRs that are too spammy
will just get closed.

