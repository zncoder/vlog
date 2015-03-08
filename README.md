# vlog

Vlog provides leveled logging for Go. It supports 2 verbose logging level, v1
and v2, an info logging level and a warn logging level. Logging level is
controllable at the package level.

To use vlog in a package, first define a package level logging variable, and
then call the log methods on the variable, like,

    package foo
    
    func foo() {
      v.V1("this is verbose level 1 log")
      v.I("this is info log")
    }
    
    var v = vlog.New()

The level of individual package logging variable can be set through the flag
`-vlog`. The syntax is `package_name=logging_level`. Logging levels are,

- `v1` or `1` for verbose level 1
- `v2` or `2` for verbose level 2
- `i` or `info` for info level
- `w` or `warn` for warn level

Package name can be either a full package name, or prefix of a package name
followed by `/*`. For example, `"foo -vlog=foo=1,bar/*=i,bar/zar=w"` turns on,

- `v1` level logging for package `foo`,
- `info` level logging for any package beginning with `bar/`,
- `warn` level for packge `bar/zar`.

A `v1` log message includes the file name and line number of the caller.
A `v2` log message includes the stacktrace of the caller.

Package vlog also provides a few helper functions, such as `Check` and `CheckOK`.


