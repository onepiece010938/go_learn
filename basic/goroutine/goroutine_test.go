package goroutine_test

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestGoroutine(t *testing.T) {
	for i := 0; i < 10; i++ {
		go func(i int) {
			fmt.Println(i)
		}(i) //值傳遞會複製一份 丟進協程
	}
	time.Sleep(time.Millisecond * 50)
}

// 錯誤的寫法
func TestGoroutineWrongUse(t *testing.T) {
	for i := 0; i < 10; i++ {
		go func() {
			fmt.Println(i) //直接訪問到i，i被共享了
		}()
	}
	time.Sleep(time.Millisecond * 50)
}

func memConsumed() uint64 {
	runtime.GC() //GC，排除物件影響
	var memStat runtime.MemStats
	runtime.ReadMemStats(&memStat)
	return memStat.Sys
}

//在Go語言中，對於多執行緒是相當友善好用的，相對其他語言所需要的資源與行數都少很多。
//以Java 8為例，執行一個Thread 預設需要分配1MB 記憶體，而Golang只需要幾kB 。
//goroutine 所佔用的記憶體，均在stack中進行管理
//goroutine 所佔用的棧空間大小，由 runtime 按需進行分配
func TestGetGoroutineMemConsume(t *testing.T) {
	var c chan int
	var wg sync.WaitGroup
	const goroutineNum = 1e4 // 1 * 10^4

	noop := func() {
		wg.Done()
		<-c //防止goroutine退出，記憶體被釋放
	}

	wg.Add(goroutineNum)
	before := memConsumed() //獲取建立goroutine前記憶體
	for i := 0; i < goroutineNum; i++ {
		go noop()
	}
	wg.Wait()
	after := memConsumed() //獲取建立goroutine後記憶體
	//計算單個Goroutine記憶體佔用大小（2~4kb）
	fmt.Printf("====>%.3f KB\n", float64(after-before)/goroutineNum/1024)
}

// Process vs Thread vs Goroutine
/*

1.	進程 Process

程式 (Program)是寫好尚未執行的 code，程式被執行後才會變成進程 (Process)。
Process 進程則是指被執行且載入記憶體的 program。Process 也是 OS 分配資源的最小單位，
可以從 OS 得到如 CPU Time、Memory 與每個 process 的獨立 ID (PID)...等資源，意思是這個 process 在運行時會消耗多少 CPU 與記憶體。

1-1.	進程的優缺點

優點：每個進程有自己獨立的系統資源分配空間，不同進程之間的資源不共享，因此不需要特別對進程做互斥存取的處理。
缺點：建立進程以及進程的上下文切換（Context Switch）都較消耗資源，進程間若有通訊的需求也較為複雜。

2.	線程 Thread

線程可以想像成存在於 process 裡面，而一個進程裡至少會有一個線程，前面有說 process 是 OS 分配資源的最小單位，
而 thread 則是作業系統能夠進行運算排程的最小單位，也就是說實際執行任務的並不是進程，而是進程中的線程，一個進程有可能有多個線程，
其中多個線程可以共用進程的系統資源，可以把進程比喻為一個工廠，線程則是工廠裡面的工人，負責任務的實際執行。

2-1.	MultiProcessing 多進程 & MultiThreading 多線程

Multiprocessing 好比建立許多工廠（通常會取 CPU 的數量），每個工廠中會分配ㄧ名員工(thread)執行工作，因此優勢在於同一時間內可以完成較多的事。
Multithreading 則是將許多員工聚集到同一個工廠內，它的優勢則是有機會讓相同的工作在比較短的時間內完成。

2-2.	多線程的 Race Condition

剛剛有提到在多執行緒中 (Multithreading)，不同 thread 是可以共享資源的，而若兩個 thread 若同時存取或改變全域變數，可能會發生同步 (Synchronization) 問題。
若執行緒之間互搶資源，則可能產生死結 (Deadlock)，因此使用多線程時必須特別注意避免這些狀況發生。



3-1.	Process vs Thread

-調度層面：
進程(Process)作為擁有資源的基本單位，線程(Thread)作為調度和分配的基本單位。即：進程是資源的擁有者，線程是資源的調度者。
並發性：不僅進程之間可以並發執行，同一個進程的多個線程也可以並發執行

-擁有資源：
進程是擁有資源的基本單位，線程不直接擁有資源。
線程可以訪問隸屬於進程的資源
進程(Process)所維護的是程序所包含的資源（靜態資源），比如：地址空間、打開的文件句柄、文件系統狀態，信號處理handler
線程(Thread)所維護的是程序運行相關的資源（動態資源），如：運行棧(stack)、調度相關的控制信息、待處理的信號集...

-系統開銷：
進程(Process)的系統開銷更大：在創建或者銷毀進程時，由於系統需要位置分配和回收資源，導致系統的開銷明顯大於創建或者銷毀線程時的開銷。
進程更穩定安全：
進程有獨立的內存空間，一個進程崩潰後，在保護模式下對其他進程不會有影響，而線程只是一個進程中的不同的執行路徑

線程(Thread)有自己的堆棧和局部變量，但是線程沒有獨立的地址空間，一個進程死掉等於該進程下所有的線程死掉。
所以多進程的程序要比多線程的的程序穩健，但在多進程切換時，耗費資源大，性能較差。

4-1.	協程 Coroutine

協程是一種用戶態的輕量級線程，協程的調度完全由用戶控制（即協程相對於操作系統來說是透明的，操作系統根本不知道協程的存在）。
協程和線程一樣共享heap，不共享stack，協程由程序員在協程的代碼裡顯示調度。協程擁有自己的寄存器上下文和棧。
協程調度切換時，將寄存器上下文和棧保存到其他地方，在切回來的時候，恢復先前保存的寄存器上下文和棧stack，
直接操作棧則基本沒有內核切換的開銷，可以不加鎖的訪問全局變量，所以上下文的切換非常快。

因為是由用戶程序自己控制，那麽就很難像搶占式調度那樣做到強制的 CPU 控制權切換到其他進程/線程，通常只能進行 協作式調度，
需要協程自己主動把控制權轉讓出去之後，其他協程才能被執行到。

使用協程的好處
協程有助於實現：
狀態機：在一個子例程裡實現狀態機，這裡狀態由該過程當前的出口/入口點確定，可以產生可讀性更高的代碼。
角色模型：並行的角色模型。
產生器：有助於輸入/輸出和對數據結構的通過遍歷。

4-2.	Thread vs Coroutine

先談談子程序，又稱為“函數”。
在所有語言中都是層級調用的，A調用B，B調用C，C執行完畢返回，B執行完畢返回，最終A執行完畢
由此可見，子程序調用是通過棧實現的，一個線程就是執行一個子程序。
函數總是一個入口，一個返回，調用順序是明確的（一個線程就是執行一個函數）
而協程的調用和函數不同，協程在函數內部是可以中斷的，可以轉而執行別的函數，在適當的時候再返回來接著執行。

def A(){
    print 1
    print 2
    print 3
}

def B(){
    print 'x'
    print 'y'
    print 'z'
}

比如上面代碼如果是協程執行，在執行A的過程中，可以中斷去執行B,在執行B的時候亦然。結果可能是： 1 xy 2 3 z
同樣上面的代碼如果是線程執行，只能執行完A再執行B，或者執行完B再執行A，結果只可能是2種：1 2 3 xyz 或者xyz 1 2 3

協程和多線程的優勢？為什麼有了多線程還要引入協程？

極高的執行效率：

因為函數（子程序）不是線程切換，而是由程序自身控制的，因此沒有線程切換的開銷；
和多線程比，線程數量越多，協程的性能優勢越明顯


不需要多線程的鎖機制：

因為只有一個線程，不存在同時寫變量的衝突，在協程中控制共享資源不加鎖，只需要判斷狀態就行了，因此執行效率比多線程高很多。

4-3.	goroutine

普遍認為 goroutine 是Go語言對於協程的實現。 不同的是，Golang 在 runtime、系統調用等多方面對 goroutine 調度進行了封裝和處理，
一個goroutine就是一個獨立的工作單元，Go的runtime（運行時）會在邏輯處理器上調度這些goroutine來運行，一個邏輯處理器綁定一個操作系統線程，
當遇到長時間執行或者進行系統調用時，會主動把當前 goroutine 的CPU (P) 轉讓出去，讓其他 goroutine 能被調度並執行，也就是 Golang 從語言層面支持了協程。
Golang 的一大特色就是從語言層面原生支持協程，在函數或者方法前面加 go關鍵字就可創建一個協程。

與線程的比較
記憶體
每個 goroutine (協程) 默認占用記憶體遠比 Java 、C 的線程少。
goroutine：2KB（官方）
線程：8MB（參考網絡）

切換調度
線程/goroutine 切換開銷方面，goroutine 遠比線程小
線程：涉及模式切換(從用戶態切換到內核態)、16個寄存器、PC、SP...等寄存器的刷新等。
goroutine：只有三個寄存器的值修改 - PC / SP / DX.

*/

func TestGoroutinePROCS(t *testing.T) {
	runtime.GOMAXPROCS(1) // 設置進程綁定的邏輯處理器
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 1; i < 100; i++ {
			fmt.Println("A:", i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 1; i < 100; i++ {
			fmt.Println("B:", i)
		}
	}()
	wg.Wait()
}
