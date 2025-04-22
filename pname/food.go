package pname

import (
	"math/rand"
	"strings"
)
var fruit = strings.Split(`苹果,香蕉,橙子,葡萄,草莓,西瓜,梨,橘子,桃子,菠萝,柠檬,樱桃,蓝莓,芒果,柿子,橄榄,榴莲,柚子,火龙果,荔枝,榴莲,猕猴桃,樱桃,柚子,柑橘,柠檬,橙子,芒果,蓝莓,草莓,葡萄,菠萝,西瓜,梨,桃子,李子,樱桃,柿子,橄榄,石榴,荔枝,榴莲,龙眼,菠萝蜜,柚子,柑橘,柠檬,橙子,芒果,蓝莓,草莓,葡萄,菠萝,西瓜,梨,桃子,李子,樱桃,柿子,橄榄,石榴,荔枝,榴莲,龙眼,菠萝蜜,火龙果,番石榴,蓝莓,草莓,葡萄,樱桃,柿子,橄榄,榴莲,龙眼,荔枝,桂圆,柚子,柑橘,柠檬,橙子,芒果,蓝莓,草莓,葡萄,菠萝,西瓜,梨,桃子,李子,樱桃,柿子,橄榄,石榴,荔枝,榴莲,龙眼,菠萝蜜,火龙果,番石榴,蓝莓,草莓,葡萄,樱桃,柿子,橄榄,榴莲,龙眼,荔枝,桂圆,柚子,柑橘,柠檬,橙子,芒果,蓝莓,草莓,葡萄,菠萝,西瓜,梨,桃子,李子,樱桃,柿子,橄榄,石榴,荔枝,榴莲,龙眼,菠萝蜜,火龙果,番石榴`,",")
var  vegetables =strings.Split(`胡萝卜,青椒,西兰花,土豆,菠菜,茄子,黄瓜,西红柿,南瓜,白菜,青菜,豆角,红薯,洋葱,蒜苗,韭菜,芹菜,青葱,芦笋,番茄,油菜,芥蓝,芥菜,苦瓜,南瓜,冬瓜,丝瓜,豆芽,豆腐,冬笋,笋,竹笋,莴苣,生菜,菜心,萝卜,萝卜干,白萝卜,胡萝卜,红萝卜,青萝卜,黄萝卜,紫萝卜,红薯,番薯,地瓜,红薯叶,玉米,玉米笋,玉米须,豆腐,豆浆,豆腐脑,豆腐皮,豆腐干,黄豆,红豆,绿豆,黄豆芽,绿豆芽,豆苗,豆角,豆皮,豆渣,豆腐干,豆腐皮,豆腐脑,豆腐丝,豆腐块,豆腐包,豆腐汤`,",")
var word = strings.Split(`的,爱,吃,与,打`,",")
func GetFoodName()string{
   
    fruitIdx := rand.Intn(len(fruit))
    vegetableIds := rand.Intn(len(vegetables))
    wordIdx := rand.Intn(len(word))
    idx := rand.Intn(2)
    if idx == 0{
        return fruit[fruitIdx] + word[wordIdx] + vegetables[vegetableIds]
    }else{
        return vegetables[vegetableIds] + word[wordIdx] + fruit[fruitIdx]
    }
}