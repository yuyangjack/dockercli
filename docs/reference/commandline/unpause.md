---
title: "unpause"
description: "The unpause command description and usage"
keywords: "cgroups, suspend, container"
---

<!-- This file is maintained within the docker/cli GitHub
     repository at https://github.com/yuyangjack/dockercli/. Make all
     pull requests against that repo. If you see this file in
     another repository, consider it read-only there, as it will
     periodically be overwritten by the definitive file. Pull
     requests which include edits to this file in other repositories
     will be rejected.
-->

# unpause

```markdown
Usage:  docker unpause CONTAINER [CONTAINER...]

Unpause all processes within one or more containers

Options:
      --help   Print usage
```

## Description

The `docker unpause` command un-suspends all processes in the specified containers.
On Linux, it does this using the cgroups freezer.

See the
[cgroups freezer documentation](https://www.kernel.org/doc/Documentation/cgroup-v1/freezer-subsystem.txt)
for further details.

## Examples

```bash
$ docker unpause my_container
my_container
```

## Related commands

* [pause](pause.md)
