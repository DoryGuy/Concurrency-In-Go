# Concurrency-In-Go
Go Pipeline tools from the book "Concurrency In Go" by Katherine Cox-Buday
in directory utils.

In directory utils_generics, is a version using Go Generics for the type casting channels. 
I thought about swapping all the interface{} usages in the basic utils, but current recommendation 
by golang developers is to not do that.

Both directories have unit tests that are run on checkin to git.
