/*
Copyright 2025 linux.do

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package payment

const (
	APIKeyObjKey          = "payment_api_key_obj"
	CreateOrderRequestKey = "payment_create_order_request"
)

const (
	// OrderMerchantIDCacheKeyFormat Redis key 格式，用于存储订单号对应的商户ID
	OrderMerchantIDCacheKeyFormat = "payment:order:%s"
	// OrderExpireKeyFormat Redis key 格式，用于订单过期监听，key中包含订单ID
	OrderExpireKeyFormat = "payment:order:expire:%d"
)
