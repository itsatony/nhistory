# nhistory

a go module to do (hash) history management in-memory or distributed via redis

## use cases

originally used to keep track of webcrawler urls visited across many machines. in general, any case where you want to keep track of unique ids/items.

* nhistory works across machines/instances when configured to use redis as a database. Alternatively, it works using sync.Map for pure in-instance use.
* nhistory is (hopefully) atomic and concurrency safe (for both modes: redis, in-memory).
* nhistory offers time-to-live per tracked entry along with a cleanup interval.

## versions

* v0.1.0 initial release

## example

```go
func Init() {  
  // you have to connect this one of course...
  // if you pass in nil, NHistory will use its own in-memory map  (sync.Map)
  var RedisStateClient redis.UniversalClient 
  var name string = "crawlHistory"
  var cleanInterval time.Duration = time.Minute*1
  var timeToLive time.Duation = time.Minute*30
  var useHashing bool = true
  var CrawlHistory *NHistory = hoard.NewNHistory(name, timeToLive, cleanupInterval, RedisStateClient, useHashing)
  
  // purely optional additional settings
  // want another context for the redis ops? set it like this: (the default is shown here)
  CrawlHistory.SetRedisContext(context.Background())
  // want another hash function for the strings you pass in? set it like this: (the default is shown here)
  CrawlHistory.SetHashFunction(HashIt)
  // dynamically set a new CleanInterval
  CrawlHistory.SetCleanInterval(time.Minute*3)
  // dynamically set a time-to-live
  CrawlHistory.SetTimeToLive(time.Minute*2)
  // dynamically change the use of hashing
  CrawlHistory.UseHashing(true)
  
  // let's use it - time-to-live is 2 min
  CrawlHistory.Add("wikipedia.org")
  time.Sleep(time.Minute*1)
  willbeTrue := CrawlHistory.Has("wikipedia.org")
  // --> true
  time.Sleep(time.Minute*5)
  willbeFalse := CrawlHistory.Has("wikipedia.org")
  // --> false - This will be false even if the entry was not cleaned up! Has checks the entry's expiration!
  CrawlHistory.Add("mozilla")
  CrawlHistory.Add("anotherSite")
  CrawlHistory.Remove("anotherSite")
  willBeAUnixTimestampInt64, willBeATrueFoundBool := CrawlHistory.Get("mozilla")
  // --> 123521233, true
  willBeAZeroUnixTimestampInt64, willBeAFalseFoundBool := CrawlHistory.Get("anotherSite")
  // --> 0, false

  // you can also manually trigger a cleanup
  CrawlHistory.Clean()
}

func HashIt(s string) string {
  b := []byte(s)
  md5 := md5.New()
  md5.Write(b)
  return string(md5.Sum(b))
}
```
