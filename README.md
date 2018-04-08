# gomvpkg-light

lightweight version of gomvpkg

```console
$ gomvpkg-light --help
usage: gomvpkg-light --from=FROM [<flags>]

gomvpkg-light

Flags:
  --help       Show context-sensitive help (also try --help-long and --help-man).
  --from=FROM  Import path of package to be moved
  --to=TO      Destination import path for package
  --in=IN      target area
  --only       from package only moved(sub packages are not moved)
```

## example

2~3x faster.

gomvpkg

```shell
$ time gomvpkg -from github.com/xxx/myapp/model -to github.com/xxx/myapp/model2
...

real	0m14.696s
user	0m33.366s
sys	0m8.601s
```

gomvpkg-light

```shell
$ gomvpkg-light --in github.com/xxx/myapp --from github.com/xxx/myapp/model --to github.com/xxx/myapp/model2 --disable-gc
2018/04/08 23:22:43 start move package github.com/xxx/myapp/model -> github.com/xxx/myapp/model2
2018/04/08 23:22:43 get in-pkg /home/nao/go/src/github.com/xxx/myapp
2018/04/08 23:22:43 collect candidate directories 683
2018/04/08 23:22:43 collect affected packages 132
2018/04/08 23:22:43 loading packages..
2018/04/08 23:22:47 599 packages are loaded
...
2018/04/08 23:22:48 move package github.com/xxx/myapp/model -> github.com/xxx/myapp/model2
2018/04/08 23:22:48 takes 4.952462621s
2018/04/08 23:22:48 end

real	0m5.283s
user	0m11.218s
sys	0m3.536s
```

## `--only` option

`--only` option, is moving package exactly one package only, so, subpackages are not moved.

## todo

todo: default move action is `git mv <src> <dst>`.
fix this behaviour.
