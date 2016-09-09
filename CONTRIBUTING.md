# Code Quality and Contribution Guidelines

## Tests

Code should be submitted with a reasonable suite of tests, and code should be 
designed with testability in mind.

## Commit Messages

Commit messages should explain **why** code is changing, configuration is added,
or new types or packages are introduced. Prefer a declarative style, like 

_Pass a quit channel to allow canceling goroutines_

## Code Review

All code needs code review and at least one "thumbs up" from a colleague. After
the thumbs up **the person who submitted the merge/pull request should click
the merge button**. Always choose "delete branch" when merging to keep the 
number of old branches low on the server.

Code should be submitted as a single squashed commit via a merge/pull request.
Only open a request with fully designed and tested solution that you would be
comfortable merging. MRs are not for designing solutions, that's what issues
are for.  

If code requires special local testing, provide a test plan in an MR comment (not 
the commit message or merge request description). Step by step instructions or
a script are ideal.

Overall, you should do what you can to make reviewing efficient and effective
for your colleagues.

## Change Log

A change log is maintained in changelog.md.  When submitting a merge/pull request,
a single statement should be added to this file to help facilitate preparation of
formal change documents. This statement should describe the overall purpose of the
MR prefixed by one of the following prefixes:

* FIX - If this change contains a bugfix or security fix to an existing feature.
* NEW - Implementation of a new feature or API operation.
* Enhancement - An existing feature is altered with new functionality or significantly changed but is otherwise not a bugfix
* Refactor - No material changes to interfaces or business logic. General code cleanup. 

## Style Guide

Functions should take as few parameters as possible. If many parameters are 
required, consider introducing a new type that logically groups the data.

Large blocks of commented out code should not be checked in.

Avoid the use of global variables. Prefer a dependency injection style that
uses a mix of interfaces and concrete types.

## Sprints and Milestones

All work should be documented in issues and collected into a milestone or sprint.
Sprints map to two-week periods of time. At the beginning of a sprint, we discuss
work we want to do and make sure all of it is documented in issues. Bugs and hot
fixes can be added to the middle of the sprint, but not without discussion.

## Releases and Tagging

To create a release, do the following:

1. Pick a version number for the release. Our example below is **0.1.0**
2. Add that version number as a header in CHANGELOG.md and write a description
   of what's changed there. MR that update to CHANGELOG.md to **develop** branch. 
3. When the MR of changes is accepted, **one team member** does the following
   from the command line:

```
git fetch origin
git checkout master
git reset --hard origin/develop
git tag -a v0.1.0 -m 'version 0.1.0'
git push origin --tags
```


