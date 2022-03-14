## Concurrency and Parrallism

You can have concurrent programs but do not necessarily do parallel work.

- Concurrency: multiple tasks that can be done at the same time.
- Parallelism: doing many _different_ tasks being acted on at the same time. Multiple workers doing similar things at the same time.

## What are we doing here?

We'll be following the course of Mike Van Sickle's Concurrent Programming with Go, here are my notes attempting to understand it. Since the focus is on the concurrency concepts i've gone ahead and copied the exercise files for the start, we'll be changing the files to show the side of effects of the concepts we encounter here.

## Our goal for this system

Our goal is to query the database with the In-Memory Cache in front of it. We would then make calls to the db and the cache concurrently (in parallel) and use the one that returns something first. We'll start to build out our main function using the book.go backing store

We'll start with a file called book.go that will just hold an backing store for our db and our cache.

## Go Routines

First we'll build out main.go. This is the entry point for the project and it contains two methods a queryCache and a queryDatabase method, the main method just creates a loop and queries for books, if its in the cache then use the cache otherwise, use the db then store it in the cache.

We'll create a map for our cache that is available in main.go, and also create a random number so that we can use this to generate our id calls in main.go.

```Go
var cache = map[int]Book{}
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
```

We add two methods that are functionally the same but the database call has a sleeptimer to simulate db operations.

```Go
func queryCache(id int) (Book, bool) {
	b, ok := cache[id]
	return b, ok
}

func queryDatabase(id int) (Book, bool) {
	time.Sleep(100 * time.Millisecond)
	for _, b := range books {
		if b.ID == id {
            cache[id] = b
			return b, true
		}
	}

	return Book{}, false
}
```

Next we'll change our main method to handle the calls to the database and the cache. This is still in sync so we first check the cache and then check the db.

```Go
func main() {
	for i := 0; i < 10; i++ {
		id := rnd.Intn(10) + 1
		if b, ok := queryCache(id); ok {
			fmt.Println("from cache")
			fmt.Println(b)
			continue
		}
		if b, ok := queryDatabase(id); ok {
			fmt.Println("from database")
			fmt.Println(b)
			continue
		}
		fmt.Printf("Book not found id: %v", id)
		time.Sleep(150 * time.Millisecond)
	}
}
```

When you run this you'll see that everything comes from the database, at first if you run it a couple of times you might see a few things from the cache.

### What are goroutines?

Its like a thread, but not really.

- Thread: Has own execution stack, has a fixed stack space. Managed by the OS.
- Goroutine: Has own execution stack, Variable stack space that starts at 2kb. Managed by the go runtime. The runtime has an interface that allows a small number or threads and allows those threads to work with those routines.

### Creating an goroutine

All you have to do is wrap a function into and use the go keyword, to make a function a goroutine. We'll go ahead and wrap the if statements into anonymous functions and prefix them with go. If we run that we see that it still runs....sort of.

### goroutines in main, some hidden issues

Recall that there is a time.Sleep call at the end of the main function. If we comment that out we see that there is no longer an output... what is going on?

The goroutines that we have get setup but the main function reaches the end first before the goroutines have a chance to finish. Program execution ends as soon as the main function exits for go. So we have setup the goroutines, but they never got a chance to finish, or atleast all of them.

If you count the number of outputs, you will see that you only have a few. You will also see that the we are not using the cache effectively (we'll handle this using channels)

```Go
func main() {
	for i := 0; i < 10; i++ {
		id := rnd.Intn(10) + 1
		go func(id int) {
			if b, ok := queryCache(id); ok {
				fmt.Println("from cache")
				fmt.Println(b)
			}
		}(id)
		go func(id int) {
			if b, ok := queryDatabase(id); ok {
				// cache[id] = b
				fmt.Println("from database")
				fmt.Println(b)
			}
		}(id)
		time.Sleep(150 * time.Millisecond)
	}

	time.Sleep(2 * time.Second)
}
```

## The Sync Package

### What is it?

A way to synchronize our goroutines

### WaitGroups (A routine waits on one or more to finish the work)

Waits for a collection of goroutines to finish. Recall how we had to add a call to wait for all of the routines to finish. We are going to take that out since its error prone. i.e. The time is not an efficient way to handle.

First we create a waitGroup, and pass it in to our goroutine functions. Everytime that we are about to start a goroutine, we call the add method on the waitGroup. At the end of each goroutine we call the Done() to signal that the routine is done. Then at the end of the process we call the Wait() to wait for all of the routines to finish.

```Go
func main() {
	wg := &sync.WaitGroup{} //a wait group struct
	for i := 0; i < 10; i++ {
		id := rnd.Intn(10) + 1
		wg.Add(2) //update the waitGroup counter
		go func(id int, wg *sync.WaitGroup) {
			if b, ok := queryCache(id); ok {
				fmt.Println("from cache")
				fmt.Println(b)
			}
			wg.Done() //call the Done method when finished
		}(id, wg) //pass it in

		go func(id int, wg *sync.WaitGroup) {
			if b, ok := queryDatabase(id); ok {
				fmt.Println("from database")
				fmt.Println(b)
			}
			wg.Done()
		}(id, wg)
		// time.Sleep(150 * time.Millisecond)
	}
	wg.Wait() //wait for all goroutines to finish
}
```

#### There is still one error.

We'll have to comment out the call to update the cache in queryDatabase since we now how two concurrent routines that are writing/accessing it. For now, we'll create a resolution for this in a later module.

### Mutexes (Sharing Memory)

A mutual exclusion lock. This is where only the owner of the lock can access the code, this protects the memory that is shared between routines.

#### There is a race condition

There is tooling support to detect race conditions that are happening. The command `go run --race .` will still run your code, but add tooling support for identifying race conditions. Here an example error that you might get:

```
==================
WARNING: DATA RACE
Write at 0x00c0000a41b0 by goroutine 8:
  runtime.mapaccess2_fast64()
      .../map_fast64.go:52 +0x1ec
  main.queryDatabase()
      .../main.go:46 +0x138
  main.main.func2()
      .../main.go:27 +0x3c

Previous read at 0x00c0000a41b0 by goroutine 25:
  runtime.evacuate_fast32()
      .../map_fast32.go:373 +0x37c
  main.queryCache()
      .../main.go:38 +0x70
  main.main.func1()
      .../main.go:19 +0xcc
```

I've removed the other directories. As you can see that it is saying that a goroutine is accessing something on line 46 and anopther on line 38. Those are the places in our functions where we write and read from the cache. We will solve this problem using mutexes.

In our code, we'll go ahead and added a mutex, and pass a pointer to it around much like our wait group.

```Go
func main() {
	wg := &sync.WaitGroup{}
	m := &sync.Mutex{}
	for i := 0; i < 10; i++ {
		id := rnd.Intn(10) + 1
		wg.Add(2)
		go func(id int, wg *sync.WaitGroup, m *sync.Mutex) {
			if b, ok := queryCache(id, m); ok {
				fmt.Println("from cache")
				fmt.Println(b)
			}
			wg.Done()
		}(id, wg, m)

		go func(id int, wg *sync.WaitGroup, m *sync.Mutex) {
			if b, ok := queryDatabase(id, m); ok {
				fmt.Println("from database")
				fmt.Println(b)
			}
			wg.Done()
		}(id, wg, m)
	}
	wg.Wait()
}
```

The mutex will get passed into the functions that are causing the race condition( queryCache and queryDatabase).

```Go
func queryCache(id int, m *sync.Mutex) (Book, bool) {
	m.Lock() //queryCache now owns the memory location
	b, ok := cache[id]
	m.Unlock() //release the memory location
	return b, ok
}

func queryDatabase(id int, m *sync.Mutex) (Book, bool) {
	time.Sleep(100 * time.Millisecond)
	for _, b := range books {
		if b.ID == id {
			m.Lock() //queryDatabase now owns the memory location
			cache[id] = b
			m.Unlock()
			return b, true
		}
	}

	return Book{}, false
}
```

#### Read/Write Mutexes

Our implementation works, but its not really that efficient since we'll be locking the code more while we are reading the from the cache (which we'll do much more often than the other way around). The problem is that we have multiple goroutines reading from the cache, and thats not necessarily all that bad. The race condition is caused by the read and write happening at the same time. This is where Read/Write Mutexes come in.

We'll update our code so that we are utilizing the read/write mutex. Here is the updated queryCache func and the unchanged queryDatabase func. In a RWMutex we allow for multiple readers to read from the cache, but we still maintain tha there is only one writer that writes to the cache, when a routine needs to write to the cache we it will clear out/or wait for all the readers to finish, and then write to it.

```Go
func queryCache(id int, m *sync.RWMutex) (Book, bool) {
	m.RLock() //queryCache now owns the memory location
	b, ok := cache[id]
	m.RUnlock() //release the memory location
	return b, ok
}

func queryDatabase(id int, m *sync.RWMutex) (Book, bool) {
	time.Sleep(100 * time.Millisecond)
	for _, b := range books {
		if b.ID == id {
			m.Lock() //queryDatabase now owns the memory location
			cache[id] = b
			m.Unlock()
			return b, true
		}
	}

	return Book{}, false
}
```

This is useful when you have an asymetry of readers and writers.

## Channels

A Higher level construct that allows for routine to routine communication without shaing memory. A channel sends a copy of the memory.

> "Don't communicate by shaing memory, share memory by communicating" - Rob Pike

With a channel the sender sends a message into the channel, and the receiver takes the message from the channel.

### Crearing Channles

We use the make function with the `make()`.

```Go
ch1 : make(chan int) //create an unbuffered channel
ch2 : make(chan int, 5) //create a buffered channel with a size of 5
```

Here is a simple program that utilizes the channel

```go
func main() {
	wg := &sync.WaitGroup{}
	ch := make(chan int) //make a channel of ints
	wg.Add(2)
	//recevier routine
	go func(wg *sync.WaitGroup, ch chan int) {
		fmt.Println(<-ch) //read from the channel
		wg.Done()
	}(wg, ch)
	//sender routine
	go func(wg *sync.WaitGroup, ch chan int) {
		ch <- 42 //send 42 into the channel
		wg.Done()
	}(wg, ch)
	wg.Wait()
}
```

Here in the code we have a sender and receiver. We use the `<-` operator to send messages into a channel and to read from channel. It just depends on which side the channel variable resides.

> Channels are blocking constructs. You need go routines to be created as senders and receivers that are available to work on the channel.

### Buffered Channels

Generally, you have to have the same number of receivers and senders for a channel. But there are cases when you want to send more than one stuff, you'll have to handle this of course in your channel, but yes. This is where you make a buffered channel. Here is what it would change to:

```Go
func main() {
	wg := &sync.WaitGroup{}
	ch := make(chan int, 1) //a buffer with a size of 1
	wg.Add(2)
	go func(wg *sync.WaitGroup, ch chan int) {
		fmt.Println(<-ch)
		wg.Done()
	}(wg, ch)
	go func(wg *sync.WaitGroup, ch chan int) {
		ch <- 42
		ch <- 27 //sender sends another message
		wg.Done()
	}(wg, ch)
	wg.Wait()
}
```

### Channel Types

#### Bidirectional

created channels are always bidirectional, that can send and receive channels. Where the channel gets used in the function signature you'd see it like this:

```go
func myFunction(ch chan int) {}
```

#### Send Only

When you only want to send messages to a channel you can type the channel as such in the function that uses the channel to send messages.

```go
func myFunction(ch chan<- int) {}
```

#### Receive Only

Conversely, you can also type the channel to only receive messages.

```go
func myFunction(ch <-chan int) {}
```

### Closing Channels

Panics are created when you send a message to a closed channel. Channels are closed using the built-in method `close()`. Closing channels are always closed by the sender of the channel. Receivers just get the zero value of the channel when the channel is closed.

### Control Flow

#### If Statements

You can use the `, ok` syntax to check if the channel is ok to receive messages from. When a channel is read from it also returns an boolean along with the message.

```Go
func main() {
	wg := &sync.WaitGroup{}
	ch := make(chan int, 1)
	wg.Add(2)
	go func(wg *sync.WaitGroup, ch chan int) {
		if msg, ok := <-ch; ok {
			fmt.Println(msg, ok)
		}
		wg.Done()
	}(wg, ch)
	go func(wg *sync.WaitGroup, ch chan int) {
		close(ch) //closed channel, no messages
		wg.Done()
	}(wg, ch)
	wg.Wait()
}
```

The code here has a channel that checks if `ok == true`. If it is then we read the message, if false then we just call done

#### For Loops

You can also use the looping construct to read through your messages at the receiving end using the `range` operator on the channel to retrieve messages. The for loop looks for the channel to be closed so that it stops trying to receive mesages from the channel. Don't forget to close the sending channel at the end of the loop, otherwise you will cause a deadlock.

```Go
func main() {
	wg := &sync.WaitGroup{}
	ch := make(chan int, 1)
	wg.Add(2)
	go func(wg *sync.WaitGroup, ch chan int) {
		//use the range to get messages from the channel.
		for msg := range ch {
			fmt.Println(msg)
		}
		wg.Done()
	}(wg, ch)
	go func(wg *sync.WaitGroup, ch chan int) {
		for i := 0; i < 10; i++ {
			ch <- i
		}
		close(ch) //the receiver looks for the closed channel, otherwise deadlock
		wg.Done()
	}(wg, ch)
	wg.Wait()
}
```

#### Select Statements

A select statement looks very similar to a switch statements, but maps behaviors to channels. If you dont have a default, the select statement can block work. We'll go ahead and use this concept in our main code to work with channels. The way we are going to architect this is to have a select statement work on receiving messages from the cache or db and do the appropriate work.

Here is the main function:

```Go
func main() {
	wg := &sync.WaitGroup{}
	m := &sync.RWMutex{}
	chCache := make(chan Book)
	chDb := make(chan Book)
	for i := 0; i < 10; i++ {
		id := rnd.Intn(10) + 1
		wg.Add(2)
		//make the two go routines senders
		go func(id int, wg *sync.WaitGroup, m *sync.RWMutex, ch chan<- Book) {
			if b, ok := queryCache(id, m); ok {
				ch <- b //send to cache channel
			}
			wg.Done()
		}(id, wg, m, chCache)
		go func(id int, wg *sync.WaitGroup, m *sync.RWMutex, ch chan<- Book) {
			if b, ok := queryDatabase(id, m); ok {
				ch <- b //send to db channel
			}
			wg.Done()
		}(id, wg, m, chDb)

		//new go routine that acts as the receiver
		go func(chCache, chDb <-chan Book) {
			select {
			case b := <-chCache:
				//message from cache
				fmt.Println("from cache")
				fmt.Println(b)
				<-chDb //throw away the message from the db, since we got it from the cache
			case b := <-chDb:
				//message from db
				fmt.Println("from database")
				fmt.Println(b)
			}
		}(chCache, chDb)
		time.Sleep(150 * time.Millisecond)
	}
	wg.Wait()
}
```

We created two new channels, and modified our two goroutines, to be senders to the cache channel and the db channel. Then we added a third goroutine that will act as a receiver of messages from that channel. We used the select statement to coordinate actions from either the db or the cache and print it to the appropriate output.
