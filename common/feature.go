/*
 * Copyright © 2021 ZkBNB Protocol
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package common

import (
	"github.com/zeromicro/go-zero/core/logx"
	"strings"
	"time"
)

func Test(feature string, functionNameConfig string, functionName string) {
	if functionNameConfig == "" {
		return
	}

	if !strings.Contains(functionNameConfig+",", functionName+",") {
		return
	}
	//if time.Now().UnixMilli()%2 == 0 {
	if feature == "sleep" {
		logx.Infof("%s->%s", functionNameConfig, feature)
		time.Sleep(2 * time.Minute)
	} else if feature == "panic" {
		logx.Severef("%s->%s", functionNameConfig, feature)
		panic(functionNameConfig + "->" + feature)
	}
	//}
}
