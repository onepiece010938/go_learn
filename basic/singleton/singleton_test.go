package singleton

import (
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

/*
1. Singleton pattern，中文叫做單例模式，字面上的意思其實就說明這個模式所帶來的含義，
也就是程式中有運行著一個 object，這個 object 只能有一個，
那這個 object 會需要經過 initialize 的步驟，可是這步驟只能夠一次，並且提供可以取得該 object 的 function。
而每個 thread 想要存取該 object 的話都只會存取到同一個 object。

那 initialize 只能一次就需要考慮到 race condition 的問題了。
*/

type Singleton struct{}

var singleInstance *Singleton

// 1-1. race condition 版本的 singleton

func GetRaceInstance() *Singleton {
	time.Sleep(100 * time.Millisecond)
	if singleInstance == nil {
		fmt.Println("INIT singleInstance")
		singleInstance = &Singleton{}
	}
	return singleInstance
}

/*
現在有一個 Singleton 型態的 struct，透過一開始先建立一個 *Singleton type 的 singleInstance，
因為是採用 pointer 的方式，所以 instance 在程式一開始運行時會是 nil 的，
而透過 GetRaceInstance function 回傳 *Singleton，並且會判斷是不是 nil，如果是代表還沒被初始化，因此就簡單的 create Singleton struct 並回傳。

那麼下一次當有其他 thread 想要取得同一個 singleInstance，就只要在呼叫 GetRaceInstance 就可以得到同一個 singleInstance。

但問題在於：
初始化這段，發生的時機點可能是我這隻程式會 concurrent 的跑很多 goroutine，
每一個 goroutine 都會想用到 singleton struct，所以每一個 goroutine 都會透過 GetInstance 去取得相同的 instance。

*/
func TestGetRaceInstance(t *testing.T) {
	for i := 0; i < 100; i++ {
		go func() {
			GetRaceInstance()
		}()
	}
	time.Sleep(2 * time.Second)
}

/*
這樣是有可能同時多個 goroutine 都進入 instance == nil 的條件裡面並且初始化，（"INIT singleInstance" 輸出多次）
試想如果初始化後裡面的 field 值也許是給 defualt 值，但是同時 singleton 也有提供 func 去對裡面的 field 進行計算的話，
這樣會導致每個 goroutine 都可能會拿到不 consistent 的值。

或是用另外一個例子來想，這個 singletion 是負責管理連接 database 的 *sql.DB，在一支程式只需要共用 *sql.DB 就好，但是如果持續地重新建立與 db 的連線會很花時間的。
*/

// 1-2. 用互斥鎖來實現 singleton

/*最簡單實現 singleton 的方式就是使用 Mutex Lock，在 Golang 提供了 sync.Mutex 可以使用*/
var mutex sync.Mutex

func GetInstanceMutexLock() *Singleton {
	mutex.Lock()
	defer mutex.Unlock()
	if singleInstance == nil {
		fmt.Println("init singleton")
		singleInstance = &Singleton{}
	}
	return singleInstance
}

/*
可以將 GetInstance 改成這樣，每一次獲取 Singleton 前都先上鎖，直到判斷完才解鎖，這樣就不會出現 race condition。

透過在裡面加 Println 再同時用多個 goroutine 可以觀察到不會有同時進去 init singleton 的條件裡面。
*/
func TestGetInstanceMutexLock(t *testing.T) {
	for i := 0; i < 100; i++ {
		go func() {
			GetInstanceMutexLock()
		}()
	}
	time.Sleep(2 * time.Second)
}

/*
但是，這樣的方式也是有缺點的，在大量的 goroutine 想要獲取的情況下，
因為上了互斥鎖，每個 goroutine 都要等待，會降低不少 performance 的。

改進的方向可以朝著，這個 init 只需要被其中一個 goroutine 呼叫成功就好，
其他 goroutine 只需要直接獲取不需要進去 init 的階段，而這樣的 goroutine 佔絕大多數。
*/

// 1-3. 用雙重檢查的方式來實現 singleton

/*這個方式叫做 Double Check Lock，也可以看成是 Check-Lock-Check 的流程，
這樣的作法是想要盡可能地減少並發中競爭和同步的開銷。*/

func GetInstanceDoubleCheckLock() *Singleton {
	if singleInstance == nil {
		mutex.Lock()
		defer mutex.Unlock()
		if singleInstance == nil {
			fmt.Println("init singleton")
			singleInstance = &Singleton{}
		}
	}
	return singleInstance
}
func TestGetInstanceDoubleCheckLock(t *testing.T) {
	for i := 0; i < 100; i++ {
		go func() {
			GetInstanceDoubleCheckLock()
		}()
	}
	time.Sleep(2 * time.Second)
}

/*
先檢查是否為 nil，再上互斥鎖，再檢查一次是否為 nil，為什麼前面的 check 不上鎖呢？
是因為絕大多數的 goroutine 只想要獲取已經初始化的 instance，那麼透過前面的 check 有很大的機率可以拿到。

而後面上鎖則是因為當多個 goroutine 都進入了 == nil 的階段後，由於彼此要搶誰可以成功 init instance，
所以透過鎖的機制來控制，最後面再做一次檢查則是為了讓其他等待的鎖的人在第一個拿到鎖的 goroutine 成功初始化後，
其他 goroutine 就算再次拿到鎖也沒必要再進行初始化了，所以最後才會再做一次檢查。

透過雙重檢查的方式提升了 performance，讓絕大多數的 goroutine 並不需要經歷過搶 lock 的階段。
*/

// 1-4. 使用 atomic check 來實現 singleton
// 前面說要用到雙重 check 的方式來實現，但是在 Golang 提供 atomic package 也可以來實現類似操作：
var flag uint32

func GetInstanceAtomicCheck() *Singleton {
	if atomic.LoadUint32(&flag) == 1 {
		return singleInstance
	}
	mutex.Lock()
	defer mutex.Unlock()
	if flag == 0 {
		fmt.Println("init singleton")
		singleInstance = &Singleton{}
		atomic.StoreUint32(&flag, 1)
	}
	return singleInstance
}

func TestGetInstanceAtomicCheck(t *testing.T) {
	for i := 0; i < 100; i++ {
		go func() {
			GetInstanceAtomicCheck()
		}()
	}
	time.Sleep(2 * time.Second)
}

/*
透過宣告一個 flag 變數，並且使用 atomic.LoadUint32 的操作在一開始就原子性的檢查是否有初始化，
如果有被初始化過的話值就會是 1，如果沒初始化過，則透過上鎖，
並再度檢查 flag 的值是否為 0，為 0 代表沒有被初始化過，
最後進行初始化並且將 flag 的值透過 atomic.StoreUint32 的原子化儲存。
*/

// 1-5. 使用 sync.Once 來實現 singleton
// 前面使用 atomic 的方式，其實就是 sync.Once 的封裝，來看原始碼就知道：
/*
// Once is an object that will perform exactly one action.
type Once struct {
  // done indicates whether the action has been performed.
  // It is first in the struct because it is used in the hot path.
  // The hot path is inlined at every call site.
  // Placing done first allows more compact instructions on some architectures (amd64/x86),
  // and fewer instructions (to calculate offset) on other architectures.
  done uint32
  m    Mutex
}
這邊的 done 就是我們前面所說 flag 的用途，m 也就代表互斥鎖，而 Once struct 提供這個 Do func：

func (o *Once) Do(f func()) {
  if atomic.LoadUint32(&o.done) == 0 {
    o.doSlow(f)
  }
}

func (o *Once) doSlow(f func()) {
  o.m.Lock()
  defer o.m.Unlock()
  if o.done == 0 {
    defer atomic.StoreUint32(&o.done, 1)
    f()
  }
}

因為第一步先透過 atomic 檢查，再來上鎖，再檢查一次，最後 atomic 儲存。
所以只要使用 sync.Once 的封裝，可以確保傳進去 f 一定只會執行一次。
這邊值得注意的是最後 atomic 儲存是採用 defer，所以如果傳進去的 f panic 了，還是會初始化成功，
之後進來後，在第一關檢查就會不成功，也就是之後都不會成功呼叫 f 了，
也要注意如果 f 初始化動作永遠不會跳開，是會造成 deadlock 的，
因為其他 goroutine 都會通過第一關檢查然後一直的等待 lock 釋放出來，其他 goroutine 都無法正常運作。

所以最終版本的 singleton 的實現方式如下：
*/

var once sync.Once

func GetSingleObj() *Singleton {
	once.Do(func() {
		fmt.Println("Create Obj")
		singleInstance = &Singleton{}
	})
	return singleInstance
}

//這樣是最簡潔也最安全的實現 singleton 了。
func TestGetSingleObj(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			obj := GetSingleObj()
			fmt.Printf("%x\n", unsafe.Pointer(obj))
			wg.Done()
		}()
	}
	wg.Wait()
}

/*
個人感覺 singleton 大多用在 global variable 的情境上，但是要知道用太多這種東西其實還滿 evil 的，
我會建議採用依賴注入 (dependency injection) 來取代單例，比如說前面說 sql.DB 的 instance 共用就好，
但是程式初始化前就應該要先連 database 並且將這個 instance inject 到每一個需要使用 db 的 module 就好了。

例如：
*/
func TestSampleDAO(t *testing.T) {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432?sslmode=disable")
	if err != nil {
		panic(err)
	}

	dao := NewDAO(db)
	fmt.Println(dao)
}

func NewDAO(db *sql.DB) *DAO {
	return &DAO{db: db}
}

type DAO struct {
	db *sql.DB
}

/*
程式一開始就會先連接 db 並檢查是否連線正常，
接著 DAO 由於需要用 db 所以將 *sql.DB instance 注入到裡面，
這樣可以保證 DAO 都會是用到同樣的 *sql.DB。
*/
