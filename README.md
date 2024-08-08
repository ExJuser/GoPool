# GoPool
基于Golang的高性能协程池

benchmark result:
![img.png](img.png)

Go 的 GMP 经过不断的优化性能已经非常高了，GoPool 由于限制了最大 Goroutine 的数量导致效果不如原生 GMP 好，但是在内存占用、内存分配次数上还是有一定的优势
