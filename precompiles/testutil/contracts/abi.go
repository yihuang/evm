package contracts

//go:generate go run github.com/yihuang/go-abi/cmd -input ICS20Caller.json -artifact-input -output ics20caller/abi.go
//go:generate go run github.com/yihuang/go-abi/cmd -input DistributionCaller.json -artifact-input -package distcaller -output distcaller/abi.go
//go:generate go run github.com/yihuang/go-abi/cmd -input Counter.json -artifact-input -package counter -output counter/abi.go
//go:generate go run github.com/yihuang/go-abi/cmd -input FlashLoan.json -artifact-input -package flashloan -output flashloan/abi.go
//go:generate go run github.com/yihuang/go-abi/cmd -input GovCaller.json -artifact-input -package govcaller -output govcaller/abi.go
