// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/theQRL/go-zond/common"
)

// precompiledTest defines the input/output pairs for precompiled contract tests.
type precompiledTest struct {
	Input, Expected string
	Gas             uint64
	Name            string
	NoBenchmark     bool // Benchmark primarily the worst-cases
}

// NOTE(rgeraldes24): unused at the moment
/*
// precompiledFailureTest defines the input/error pairs for precompiled
// contract failure tests.
type precompiledFailureTest struct {
	Input         string
	ExpectedError string
	Name          string
}
*/

// allPrecompiles does not map to the actual set of precompiles, as it also contains
// repriced versions of precompiles at certain slots
var allPrecompiles = map[common.Address]PrecompiledContract{
	common.BytesToAddress([]byte{1}):    &depositroot{},
	common.BytesToAddress([]byte{2}):    &sha256hash{},
	common.BytesToAddress([]byte{4}):    &dataCopy{},
	common.BytesToAddress([]byte{5}):    &bigModExp{eip2565: false},
	common.BytesToAddress([]byte{0xf5}): &bigModExp{eip2565: true},
	common.BytesToAddress([]byte{6}):    &bn256AddIstanbul{},
	common.BytesToAddress([]byte{7}):    &bn256ScalarMulIstanbul{},
	common.BytesToAddress([]byte{8}):    &bn256PairingIstanbul{},
}

func testPrecompiled(addr string, test precompiledTest, t *testing.T) {
	contractAddr, _ := common.NewAddressFromString(addr)
	p := allPrecompiles[contractAddr]
	in := common.Hex2Bytes(test.Input)
	gas := p.RequiredGas(in)
	t.Run(fmt.Sprintf("%s-Gas=%d", test.Name, gas), func(t *testing.T) {
		if res, _, err := RunPrecompiledContract(p, in, gas); err != nil {
			t.Error(err)
		} else if common.Bytes2Hex(res) != test.Expected {
			t.Errorf("Expected %v, got %v", test.Expected, common.Bytes2Hex(res))
		}
		if expGas := test.Gas; expGas != gas {
			t.Errorf("%v: gas wrong, expected %d, got %d", test.Name, expGas, gas)
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}

func testPrecompiledOOG(addr string, test precompiledTest, t *testing.T) {
	contractAddr, _ := common.NewAddressFromString(addr)
	p := allPrecompiles[contractAddr]
	in := common.Hex2Bytes(test.Input)
	gas := p.RequiredGas(in) - 1

	t.Run(fmt.Sprintf("%s-Gas=%d", test.Name, gas), func(t *testing.T) {
		_, _, err := RunPrecompiledContract(p, in, gas)
		if err.Error() != "out of gas" {
			t.Errorf("Expected error [out of gas], got [%v]", err)
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}

// NOTE(rgeraldes): unused at the moment
/*
func testPrecompiledFailure(addr string, test precompiledFailureTest, t *testing.T) {
	p := allPrecompiles[common.HexToAddress(addr)]
	in := common.Hex2Bytes(test.Input)
	gas := p.RequiredGas(in)
	t.Run(test.Name, func(t *testing.T) {
		_, _, err := RunPrecompiledContract(p, in, gas)
		if err.Error() != test.ExpectedError {
			t.Errorf("Expected error [%v], got [%v]", test.ExpectedError, err)
		}
		// Verify that the precompile did not touch the input buffer
		exp := common.Hex2Bytes(test.Input)
		if !bytes.Equal(in, exp) {
			t.Errorf("Precompiled %v modified input data", addr)
		}
	})
}
*/

func benchmarkPrecompiled(addr string, test precompiledTest, bench *testing.B) {
	if test.NoBenchmark {
		return
	}
	contractAddr, _ := common.NewAddressFromString(addr)
	p := allPrecompiles[contractAddr]
	in := common.Hex2Bytes(test.Input)
	reqGas := p.RequiredGas(in)

	var (
		res  []byte
		err  error
		data = make([]byte, len(in))
	)

	bench.Run(fmt.Sprintf("%s-Gas=%d", test.Name, reqGas), func(bench *testing.B) {
		bench.ReportAllocs()
		start := time.Now()
		bench.ResetTimer()
		for i := 0; i < bench.N; i++ {
			copy(data, in)
			res, _, err = RunPrecompiledContract(p, data, reqGas)
		}
		bench.StopTimer()
		elapsed := uint64(time.Since(start))
		if elapsed < 1 {
			elapsed = 1
		}
		gasUsed := reqGas * uint64(bench.N)
		bench.ReportMetric(float64(reqGas), "gas/op")
		// Keep it as uint64, multiply 100 to get two digit float later
		mgasps := (100 * 1000 * gasUsed) / elapsed
		bench.ReportMetric(float64(mgasps)/100, "mgas/s")
		//Check if it is correct
		if err != nil {
			bench.Error(err)
			return
		}
		if common.Bytes2Hex(res) != test.Expected {
			bench.Errorf("Expected %v, got %v", test.Expected, common.Bytes2Hex(res))
			return
		}
	})
}

// Benchmarks the sample inputs from the DEPOSITROOT precompile.
func BenchmarkPrecompiledDepositroot(bench *testing.B) {
	t := precompiledTest{
		Input:    "b755417ac7b0a00d7a04ccc9ba74c5bf46213704ae6c366176b5c92dd6c209331a23b29e4c22f8db4b4c8f90c69a6e6a14c0ecae5abde6f6e6f03a41154ef97d49f55ab23e1e421f7dba88182c8ee507cfa225ddf6fbb7e5d5331dbf21995313bf40d127ca8889ce06fd10b848f83bf6b0b7412ece6255b895833c17309c39af542eb6a12c61aa788dd1dd054c8c6630e80e34c211f107c7c440342f3a434698d6abbc68bb80e98694f720e996872bd049ab12d263ac63fb63ac100c5122e1b69ae2bec37850c38f9db50928facb0a7bb39a4205eb1c7f2fdabbeb32edb17eaa9fcd68f43eb56c66e829ce95bbd86545e22f138df8b48816543622bfdb0fb36079ac56cbbfeffe6b33781c410f51dbb2188d47c924e0d4c2c8b55cf37e749d28096df5cf12256b0d62324140c4bedf3fdf75360c6a00e441a77e57e5f26e9a5786adb75b10d5d8265038eca3fe6f52568ca4099deac108604f932f793eb6e8962ec4372a49e288c488f94096b2fbc32365be7c24588dbcf83a963973a3f1ebf151fe3a36dad00bff7e72af525bfac2422a0431988289f10530c8d069c7dccdb4ac5b68981f7909f15e5faab4546d53855ef12de6cdabc1006ded7779dd91f0615621b9f7b2a5945dd044504e7fdec80c62c68b16179ff628f9f6c59398d9dfa35fcc2a40022f9ed19cb726c3052076502ac5b34ae945065191a0161526e750942f4a4a7b05695fe7f3cc7f4c22a7254063f78d80f5da1d9b737a642a59ad16d955e118a1b1e68e7fc83d198780cb37a84da1095a96c1bef28e3cb8561c4eaa05647a523df7b1d39695cd5a47e51e4e75d865dadefb2933404f684422400de0158c1a8f2dabc8b87ced747de591a68bd0f9aadbf90911cdb6b08f77450454d005129c10a9be844be3ad848b0a674356c0d31b9f19acf45a9781e85bf1a6a2b88999313bfaf816637b239d4fced92293082f649dc3c48ff0443505e63dd948f53dc867fbb523faaa9b6f95e6193b18b0bf3150b35564acc654ddcc6c8122fb134c4acb45a6f583ec5f8374b5b118153271aef87b4986d76f2e6996523cfab3dea3f794fe62c1ff5736fdf9d13295c8d3b29fddc06200aae6383aa3b3e9581eb48131d27b834c0e903e9d1e553afb56199ea789fd475da516b428d655ac86662dbb26a85da97af5dbf166a31cdb6d6dc0cec8137d75ba3e892a3217beeb4e3de845f2e762ddfd7ad1c1cc846dfbf0b75590268347a6a5566599352661e25149058957b0363e82e0e6b707d72d2ecb4f7742333586aaeed28a52f905a111abcce4fdb3950b6ee09e59ef42d5e156315ef6c3a0b3de5d0643b5c5b020eeed54d2a8e619fd15e35ae182cda8415292e40e52af5a6434187af39ee73a5ddf70d3db69de5c54cc24d1e42071f39ef150803d37550a34a4de0bcbfb02e668f7323ac7293e0d743a383d9d6120df38db45b38542e0eb3fb2d73339d523cc2f2e35f6fa2de260de4c8191a242a67cc08f6ee7ed4546c8c2c8883516f1c21e5432a56850f7c1ab67fda639424d5d5ab5ae5fa6b54879ea669f9933874077936d9a6023ef21a270adfe2ccef04ee076773e0aaa09af869dde0b2db2e7a2ee9172b5b39510e6bc24e8b4efd3c80162af808bf8fbe42ef4d2a222f825f64ea313db470905f93cd3bd944773354645845880eadab41c47f2a40abc3d1e1b1095e601a11ef1a2e00edc52b500c9cec70fea2d863bc7182f276bf0ca8b9c3bbd4e8cbff8eefc62b8bef1702681967d11c3178faa42baec7752f3bc9837b9e7b2afca131ed01c8cb54465c79261d4a7ec9407c1779457f1e5af2a76d98f0e589f70fcedc1b6a612000a4224d5dc3abe7197485c5b81e6cda403b0a8e974e1ea3c0f327f55a87a7ff063b0391ecdcd57ecef81d3f66063223fcad66fdf79c9f4bb823cf52aa9f7f141bf76ba05e6a8d2e5cec872a59285a7e90dbaa2c931a26979c60f6060e2f5b458f44d16c637a7d4508ca233008232ed6aa9c0557f2ac51fe706abd45e04f5b812b19f427870343c323bb1e7a98ea7fac273444b73777956b84dbb0de64d623211598a7d1ff169c4445469a4b0e50677ed828f60ea4f028b2aed7bf5e56a6ae7607ed6c9b291a3c3d76f1ea5a0419f59cb91c89f07d9706ec46c42fabd47b280c7685355f74b58ac10599e927ff63d5015f2a8e5471e3b5afbd208fa761f8106f9293e821d483b1da1b298a386ad82459db140ea87e1a3a23a6426bd814385a28c4c3a842a44e2024e486b0b4644658d73236ceed421de204656a11182f21e94d66ea9c290249fc9e374cedd4e67b0969acfc1f4ecbdd401ec936353309134becd14db6784a51283bda7980292cde6c8cd35ba0b3a1109905ab8b8a2f90e93c6cf47dd16737894d04426c9e3b85191debeb9302511faedb31a21ab3b22a90aec1e22985757e6ffccbe751f9c6256d69bad6d6e1cfe989202e87cf2d54f1b4e4163902d6e0df22f3accac0e28cccb4c0ed17652bdce8c5aaa3160ed20ef1bd455119e00e1c3832e6c838a6b7635723c4e078b8d2df66f9c3ab7999cbb6dea42f35ab3ea13b6d0a0a673615337da54f2ab1bbfbf3b0912fcc1b73798a883281a8a831938c02e226533b38d426204ca082e9a39ed8736d620465954ffd5a5645aff6684702a08569534713ce8b9be795a979a947f67225a2dda27d20e4f906e39738a413233ac8b5c7da0cf59caa54ac89b7002a0b73997ce1879949f3f5d37a82f49047e6c6487351eaa5e7a8145120fa77c2383278ae7f7b2b169b64c8a4ac6e6e6ae8c78b5e755d42e24621ee59479c0603736647480f944c90545038ba8fb048dfd475c739fec820c792010b4e9203d1c18b908e9feb448caa815bdd8e8363ba793544ce67ec4ecdac3b0cea2acc4d51aa5fc2f38168f4c9c83386e9c5fc2e96fcab0a9827f6daf8eb1fcb8c391a7cbbbea098b6946ea5111f63f1a31455d49a45bdb48f999a3153ee37b12917e564778f7c574b3925f12a03190b21b8bb062cd7663c377a396cb99cef834eea667d19ed20d22f11e8564c3cd670fd3b020e764118cc22cabf2f967776b0406cc67f7510e10b35d7779031822b6feab45735b7121d628160783c40f3e3cc2cccebc623d77e8f4d3c9de88fe0660a137aa5f5811c873ad57115200468aaac22a768f8e227c95faa7324b1d1a81b03c81a76bf6517a9645f5f53a8f5865ea278d95d675b25c1def5d818fbdb2a7d497bc3a0645d4e362ede1b4fbe472e6f4968d92d30ed475e55205fedbdcf2ed46bc7386d05812a7cea540aa6544525ae423bfeccad82a6d1269c9c19b8c58ce6a1565d0dc54853d32962912f3c7508276e7189686ff49436dfd80c36d9a4d8b41e0845082ac538db51b5b11e2fe23756c2360f1dc4fcdbd7d2ea2e714795baad4c7aba58ac00566b52c36951be76efbf083628a20fdd828a15e0781faadf32262f09f017975dc36b9639bb705fb9e0321a05efeed7186e96d3004b7238efc015503aa15d81db812a9876870564d2a81632d30d4e09ad3b04b8cef2801c3a095dcbfe6b321f187a55a57eb42ae495d89a9fdecb91e3795c24f57f4b45d050847462505ba32e0de293f228d7452ef8cdd0495b4906047473a38a900d471930b29c1649829b030c02500531a76174c851ad6789aa125b4ebeba72e44221b43b67e622541496bdab0790080ca396124000022369b6469269c72755f4766145f1a49baeda92f2569ef1cb0321cf9be3618fd14426e75a371d9fd30e536c4c54e47adf8e058f0c7d1036b9f42d0ae8665f1970b22fe96063025e9900c6d0d0fa8f896e5d0091e0bdc5e4c2520819d2797e9c3344e9e4e597ec89fb5f1857a9e29dcd2683f9b31eb0587a32ec3ff790bcfddfe16b32f832a68476713c80323865f95afb0f435f53e6dd609ac15289116103f6b12643674d3bcc1912988f8cb0d3610403f87eeb62b9f9485924b80f83e3dd827abbdc3fe13235b3859f6ef6662f1c69c41ddc4d705fd8a3026a294a4ec3fff4cbbfd5998e155c7f42e9b07b9eef56df0bed359f9fc550b4034bc3a7dc9960f846ca22811ed2a16b2aa44950f9ebe1af2d9a3d54ab963452542d798a087a79e2b65613aed301f46a8265f3c27898ef63211f4a4a915b40c0c58d596993a2dbdf5726582c399d07f72e009e61168393d666df03138cc1206bcebf23d636032c12e1e672d76590f7f82342df80710f6249cefd9610c4d8126f3ed181d30878f18553249227035a7fd9ef5dcd41b3236cb63c9a17097a322dcf95125b4709a1636d0a85a205677456c6404f7e59550a4d1dca12b16bac6b4c70d309d1a6390798174b71569b1e554025ba15814a8b78a135abc0423c1e3895c1851a3774a56ba4c8659346fd4d195df771b3ed12102e3b08dcde78df838f13fbdeab9d3b97ae4bdbfb00ef5fecbb65c8586bb05f49b560e6cfa63b00b65d8943bff21db171c9e4f5a1313309ae2680e0c40d7843d8e427ae180d17d3f2100a5671bd36739ef3c49020f883e9291da29660931d73a76e1ec987ed6647b8322a5c2e14380631eb9502cbf7ae0c5f49a68783a9f782f336cf7b43b6ec880e9bbbf94e51de0bbdcdd8e32725c96d853be884a83064012833252666d37b901dce7151735b57bfd5f0d51172cf43bca84b33d75ac257484d6b1fe4a1cecd2a455551fda5873cb1a952997e63185abfc8a7dddbf037e6c16ebc7baf7c7d5cb5890230631c701e503d2215ed3f3a5d41adc5d4f31605a3c08eaeb8be5696f9fd4b399096c2d8b09ba7545b35264cb46e226debf67c1e3063a3e0680e9fc45681c7bb63d84b00228d9c687e18f8698489e5035c72d3df866e05968a98adcf214f57047a4b9a8b2928d599805d6b05071e8760cf95f1cf80061048d2d125f71490dd4659f71ebace99857f17d5cd44a59294453511e0096651091c15a169fba80a082d0548ca05c24d3ae7d7c21d2a425e68f602bda9e1c367f1afae2b980e206c75f8a36d4c59e0a8c7cd4925828a126cd6f2367d3d61084a007b34a50b98bcfdb986e8547a8e8d2ee254e1546f896f067fb6f87a8a690222eaad2954f3f76e173f759b60d941828d6a6eec09ac075ddc44213fdb6dfd2d6d5ed335a4a5ba726b10147cb730ca1b17b1383021b19edf241099b7cab5761d6abaf4b6fcdc3e01069323bfd5185c4add567659c9f6db2d81f1df58b5119d44aeda1ee1daec6ba185d124907bfab72fc571cb6bc9cceea58f21279e979113eeda4687bcbfa4aff301e0ce674ce728a105a20094a6a21fc2a3735b6717f62e4e1b6f325c8bc1b039a4cda0bcf6890ec4d78ae5226a5df0772ba67f8e89d9d731bee730ff009a171669fd52051f67bba709a661c4456ed1b124c76794ae62cf569751f5c743d63babd0d6373e3b43cdd6b551b7a499b52b19940771d3fa8ac77fde46d97724e3585ccb189ced10550937a90f82d73ed06d30df7fe9a4226075dcd51f443f10cef4575e67b348c12611a4e2e0750beebabc6fbe16f05a4c055101deac46f88b833b8d28be6d7a3e4d65551e4e7fb063eccd7dc7e228f7a76aa7a771ab318cb1e1496087a0fc291a03afb243db08b0c2d8046ce9ef078f71e92032c9571528265bd14b1752ebb1b5dc4588bc89734411755759f820baf656764b2e53b6438f9808f82f834d9111bc4f9b1b679ff69e7dc3aaa500c411b848d7759594fe0dc2be253c5fa0d3d6dcbde9815ce7cd3e7a8e7c3a4a43be2d76ed9ee2a3f1da1a905b75bb3524603298e3410baeec49d7140a66b18da1f590b7a8cea554ef1764a071362994f07759cfdfbb235b12f9244b08a0b3648bec2c5dd9d108d966ec8deffe5ae8aebe5eb3f6d6660b7bb65f8fb06dd1f0a021812dc145701eea4e4066442fb59342488329a22c08900c7705952fcf2d3d927deaf0bc94de339e8d6c5c64970de6001a8fdb213c7d2235c80408a1834caa322c238184597969a34f40e76b0753b72c2cbfe07cd9b3ece7f9e65958e809ff53dfbee41cd28618f274cd7244d99b7982f4c4e07a24c7f08199822cbc7bc5e44cb2f03043e53375e5e3a992621ac2d6730ea75d5d76c8854ba362a33d62d1b218eaa92c84dd0c78142545ca76ab3630cf6c96add59fe226e37151585802647fc046c9306d10211d054c331f7cca6e12fc7ad5355cd3eea7cb590dd839f46666c9043663dbc904a0a0129fe5d238a15bffb28cb8459f34b62a63560ad693b693613abb4b0157f799ca2a1bef061ea99bb0e36bbb3f05e41c0b1b26f1f192c95791577194784b7a9f1eb31651e803b001e959ba4794fbcddc97ead97880b392a264c2acbe5783eec9bce148ab8d66d820c37fad69db8d9d5efed3b16e7f269d9be8a189c23fc17048d787446763a6f55b0aa140bdd14ebcf108b04ff5d4cbb27011e69d8fa00a0ce7846432f01da3ebc897c9490c057e5fa571be76724b95ac4835e2e4bfb72a61dddc08edc8f49b44d9d9f6ca2431edb2f76ad940b92ef8636f26fd60cc0128cc59dc30cb48ae72a95b4ea18cb8a88e2ab1960f7a9b6967a79696cb031b88d2d83591fd1a02107b4866f4473c0ce7d649dcbae808aff5b5438df74c82deaa8c396aa6dd4519463086efaf5b676225973b26aae176cf0cd378e8df05647d8ddc82406a34311d63eec18a5f13c3c3b41d2718f1c8a373f5fbba617ccc486f13d0468acac1923008789fb46a64c9ec68a22378ff9fb532dde12fab7a0a185a5f37eaa8ce10f7b972b5f49b70ef42fe1aa1ba087ae3204b169a9e9e0f3ff0b2827a68adb23735ba4dcb80dd2115d1de9ada506a588c7029f5a3cfeacfa1e309b7153c27dacd84f3550bc447e513f1b760101446a99b719cd4cf90d25a6d208e2c25ff564af4b7c220209a9ce02bd8e0d6d20f9c54bff3cda614e3f2fa5b35b141c6413c2dad105c135e7bd830b8ef57ebcf6837cc314571d9c207e12343ce233c6eb5b7211a1885dd760ef614facdc35430301e1a244c0dad0133bc3b8d671971c783da52f0c0d37db430efcdc6d037f3fa5e59446b8ef271ae011a55b4ca3a8cb334430d36015e54a1d7d4fef2f1057d54da4951d6b500a78572311c348e8d14504f40e20faf11d09351886c4b20052ba5f4b2ff89829164aa623811bf05fb8363c65f9d19c4719385c05e10ba9acd21510bed42ddca66c470a1153c54c32724b0c2557cc8a1ee186e4d4d78ded99a381942ae7aeb63185cd7a9b37395056ef389733b7b551acbffe1929e695c8e928af9c472e0d73701077bb84af5700d2013a4a44ce31f0e2c9a0c11fc88ef24876637d350dba6413e1bf1e3463a3a8f6f42cca9cb01520b0d6c21d00b2b89d78a21e6e45b0ca16a7170e46e86ab6596d2cd320e02a3f944a060b7d8e77541edca3816523181f59f845d88f3c20a45402dc439fc4419d5541c30b4cd0a487fd330a5a637aa363231c708a5c554e7deaa4a3aeb7cdc0a58897cc9be4b8dbeac7bb535c237fd16f29e0e51774a82219f3b910022938dc7da7257ad17dd3064e442d74ab646ebe57f617256e178cd8d185313c8f724fc46be4081dc83998a6de09e73c143e0d49fa989a3cdcf4c863adde76d85165d9be82979c81851087d99b4ac37c6a8dd9e8045a8087c55bead5cf1b0460538c06bdb1ae98d310bbfbb10b059451674f84634b6b0dc2a7003bcce323f44e964081b13f87a17d883b56dac91a7afdfc55ff9be494180a7da334ac6462d1815ac5f0804fc43f529030b7630f8f8c0ec87fc35dea02ad8cef44bfdab3ddb09793633bf5c678f600633820fb135c047cc459c91502be3b4d042438c4bf41c3f409f39888024ab6e957997a917b86b4ad883a7dcb2c8ef2c1ca74ca9a39d950589e5a22467850d349cf6f8ca5a199861da49b9cfa523e037e9d6b1ae235407d89eb726897609e761e49f295ec466beb6f70bd19b077352d47fb57b1fe6faa41662f1a633412f3d81d404c6b2105cce4d4c45451591037cc532de05d9f094827b7bc68d03238595a6fb949a392b5b110f24243921ae7762e60d2e0976c264f64d5c7f2feab800362e2d88b5a25b60f360468cbe9ef9f47ca11ee7cacf46477630ccb5a6e963271542bebe6e9c24d94625334a04ff72d3c17263e62a5e9c6bf2ce24e94ae4414eafb714deb72ee4cf25faee6c83322dee45b9581b6574d3736dc5e1ef51649b34687d0180bd7e8698e87c2594d1c15fef6ff0117dcba3d6caa0a7593fd31471ceeb829c24df70627cee3b60c6f910894e740999dd152efbf9bb120535994c1a180a448154e962bc02f638f505b0ae59105c38aaaed96b5731382c42725711483ce198f3287afda7f947d62aa2e5eecfa5373e0ec22ab7cd05128d6e7314b1f3eb5a83e020492119a254e309beadcf3d683066e8399042a8f6592ed143c62e2fa8c872918485898b0a84a3e73a51abddd7faff733e45fbe7a18b881e73b1e980a34a59577576aec2e9ec45495e7315db42f24310ef1bb2afe2f6d0fc71b0ce8baed3b1271008438f80866ed6eb1a10cab391965a4cfbfd5894976aebf00a7fe90293835500c5496d0c1452f85c34ca513c3fb16f70f41a123fe791fae36eb4e9b4324a3518ac66c0091b4875c654f66130751a54831ae446162dbbbeee997f60732967ed25fa6333087bf72beb4040a21ac983bed12d38a04c01461dfb098f0220dc8a7978385d5714c065160c7a0a1535ee23003aa31130a3e1437723177ba69c2975343e68248b164ab4a56581130e80e39de1ca4d2a5a1769821bb5824fe9a85a8417169e93ecc187ada2d1970abcac8d755f6795fbf158a95f03fa62e75e7649b76454664d41eaa5f39835e4d570d68e5ad99c8bc4a39104ce5b0cbf875995f391b5c3da5b94ed7e6344d378790f08767d1d52c970168b3282e5e33e0326c8839d762902b7bbeae532596329398ca7427aa2d30fa945e0dd7237eba0fc0d2a4199c33d8c7f83ed2302d4bc9b730b5ed843fccaeaca516e8861cfdc9b7b92f317c238eaae1bec3435cab0876bea4d6554b38991d20cb9931935ad55f5bc3792249c510a5542be5d45caac4534fdcc22b2aeada0514db450be6294a40a8b3cad31f7932526a706fcda5cfc00c3ff7587b5764f76445b8d69cb1243ff80e8ab54ce28ee1766b4df48c86fa7f888e872469da75229c8e4a8f31fdf396bfa79399362a8aa33e6d692d2fced8fd2d77a401616871c00bfffceb47a94b447393f72b4e3c1ad8c7aa3e35a727feb4616cc0d716f7f2e07794ee292c95b2bb35a388272ecc60eb19b858eb29fbdf73332df3e81fb296fccdcc3d42c3541758d37a492f9a69887893fc8243dd808fc7d984fab144fa57216e80dfa405392c7f9930c56a6a5a7ab8937478be4a036c1f805ace9fd39fa0b64858fc3a9b18c6eab87ecee6772d0e51c7d6daa0a1460dd9c8aaee100fb26c38589a8d70b109bb647ff4836831a09d2dcec9c3a42b7ae700c543b6da68ff8573be2ee942643124849aceb897a5c212458cb8845555b49fccc9ccc2eca8b7c481308fc369946ecb5ad0befbd9a157deacba14076c665c56f872af60bac27ce171771a1237fc2de03761597604c6a9b0dd5d1deb80da88586742274f34df45a37f9bfc97d2d2fdda68b4cbe6dc3eb24e93f8c31e62db46de831a0cd19310d430ffed77239bbaac135727bec62429b92094493bef5170a80fbbc203de5948cc9270ecfa74d96702cefa4a2efe9bbfa68323b31030989d8e4efb4d24052e8d5f7f63703e0f128507612216a9fc47a86345ed04c39b1f2b7f26660278bf1fae35561ac137b0e95d9b0d57b73263aee0b2cef8daf759874d8fe7595d758816863b98dd944b6eb09d7803546755020958573c5f9cd6d38291e2b5ebe1e9a575f8b37945b5a109005945357cb07562dab004cfa6ebf1ce72599709e19aed5d7ecc2afec88523003a63f3e89f5d232e480b6e31c34ac72c3e89d0cee41b353cdb161ed7919741e9447960d58b71c0a012760dc2942d62432beaee4aaa04033eb0c23d03a1a7110f7af94d9dd0290469c43d0f96dc1b1a90a49a870ac80ad84d0e6846083f3d01141f45afee3d509fef010c34363d568b99a8fb38415066759dcbe5485e6d8e97a1b3e3ec0002246ae4ebfd0e203b414c7f93a6afc4e4203588c3c5c7dfe2000000000000000000000000060a141c252c373f",
		Expected: "862581de9cddb8c039878d5a6d83409556361fbdd6ecd17e62df68c377db2362",
		Name:     "",
	}
	benchmarkPrecompiled("01", t, bench)
}

// Benchmarks the sample inputs from the SHA256 precompile.
func BenchmarkPrecompiledSha256(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "811c7003375852fabd0d362e40e68607a12bdabae61a7d068fe5fdd1dbbf2a5d",
		Name:     "128",
	}
	benchmarkPrecompiled("02", t, bench)
}

// Benchmarks the sample inputs from the identiy precompile.
func BenchmarkPrecompiledIdentity(bench *testing.B) {
	t := precompiledTest{
		Input:    "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Expected: "38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e000000000000000000000000000000000000000000000000000000000000001b38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e789d1dd423d25f0772d2748d60f7e4b81bb14d086eba8e8e8efb6dcff8a4ae02",
		Name:     "128",
	}
	benchmarkPrecompiled("04", t, bench)
}

// Tests the sample inputs from the ModExp EIP 198.
func TestPrecompiledModExp(t *testing.T) {
	testJson("modexp", "Z0000000000000000000000000000000000000005", t)
}
func BenchmarkPrecompiledModExp(b *testing.B) {
	benchJson("modexp", "Z0000000000000000000000000000000000000005", b)
}

func TestPrecompiledModExpEip2565(t *testing.T) {
	testJson("modexp_eip2565", "Z00000000000000000000000000000000000000f5", t)
}
func BenchmarkPrecompiledModExpEip2565(b *testing.B) {
	benchJson("modexp_eip2565", "Z00000000000000000000000000000000000000f5", b)
}

// Tests the sample inputs from the elliptic curve addition EIP 213.
func TestPrecompiledBn256Add(t *testing.T) {
	testJson("bn256Add", "Z0000000000000000000000000000000000000006", t)
}
func BenchmarkPrecompiledBn256Add(b *testing.B) {
	benchJson("bn256Add", "Z0000000000000000000000000000000000000006", b)
}

// Tests OOG
func TestPrecompiledModExpOOG(t *testing.T) {
	modexpTests, err := loadJson("modexp")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range modexpTests {
		testPrecompiledOOG("Z0000000000000000000000000000000000000005", test, t)
	}
}

// Tests the sample inputs from the elliptic curve scalar multiplication EIP 213.
func TestPrecompiledBn256ScalarMul(t *testing.T) {
	testJson("bn256ScalarMul", "Z0000000000000000000000000000000000000007", t)
}
func BenchmarkPrecompiledBn256ScalarMul(b *testing.B) {
	benchJson("bn256ScalarMul", "Z0000000000000000000000000000000000000007", b)
}

// Tests the sample inputs from the elliptic curve pairing check EIP 197.
func TestPrecompiledBn256Pairing(t *testing.T) {
	testJson("bn256Pairing", "Z0000000000000000000000000000000000000008", t)
}
func BenchmarkPrecompiledBn256Pairing(b *testing.B) {
	benchJson("bn256Pairing", "Z0000000000000000000000000000000000000008", b)
}

func TestPrecompiledDepositroot(t *testing.T) {
	testJson("depositroot", "Z0000000000000000000000000000000000000001", t)
}

func testJson(name, addr string, t *testing.T) {
	tests, err := loadJson(name)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		testPrecompiled(addr, test, t)
	}
}

// NOTE(rgeraldes24): unused at the moment
/*
func testJsonFail(name, addr string, t *testing.T) {
	tests, err := loadJsonFail(name)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		testPrecompiledFailure(addr, test, t)
	}
}
*/

func benchJson(name, addr string, b *testing.B) {
	tests, err := loadJson(name)
	if err != nil {
		b.Fatal(err)
	}
	for _, test := range tests {
		benchmarkPrecompiled(addr, test, b)
	}
}

// Failure tests

func loadJson(name string) ([]precompiledTest, error) {
	data, err := os.ReadFile(fmt.Sprintf("testdata/precompiles/%v.json", name))
	if err != nil {
		return nil, err
	}
	var testcases []precompiledTest
	err = json.Unmarshal(data, &testcases)
	return testcases, err
}

// NOTE(rgeraldes24): unused at the moment
/*
func loadJsonFail(name string) ([]precompiledFailureTest, error) {
	data, err := os.ReadFile(fmt.Sprintf("testdata/precompiles/fail-%v.json", name))
	if err != nil {
		return nil, err
	}
	var testcases []precompiledFailureTest
	err = json.Unmarshal(data, &testcases)
	return testcases, err
}
*/
