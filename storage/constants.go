package storage

const (
	TOSStorage            = "tos-storage"
	S3Storage             = "s3-storage"
	DefaultStorageSeconds = 14 * 24 * 60 * 60
	/*
		Tos 存储费用:
		  - 存储: 0.1元/月/G
		  - 公网流量: 0.5元/G
		  - 请求次数: 0.01元/w次
		  - 按每个对象(100k)平均取回100次计算
		  - 成本: 51.1元/G
		Token倍率:
		  - 1倍率: 2美元/M = 2048美元/G = 14745元/G
		  - 0.01倍率: 147.45元/G
		  - 0.005倍率: 73.73元/G
	*/
	ModelRatio = 0.01
)
