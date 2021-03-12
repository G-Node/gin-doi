package main

// defaultLicensesJSON are the most commonly used licenses for DOI registrations.
// The content is used to check the licenses in new DOI registrations and warn
// about discrepancies between used license URL, name and license content.
const defaultLicensesJSON = `[
{
	"URL":   "http://www.apache.org/licenses/LICENSE-2.0",
	"Name":  "Apache License",
	"Alias": [
	"Apache License",
	"Apache License 2.0"
	]
},
{
	"URL":   "https://opensource.org/licenses/MIT",
	"Name":  "The MIT License",
	"Alias": [
	"The MIT License",
	"MIT License"
	]
},
{
	"URL":   "https://opensource.org/licenses/BSD-3-Clause",
	"Name":  "The 3-Clause BSD License",
	"Alias": [
	"The 3-Clause BSD License",
	"BSD 3-Clause License",
	"BSD-3-Clause"
	]
},
{
	"URL":   "https://www.gnu.org/licenses/gpl-3.0.en.html",
	"Name":  "GNU General Public License v3.0",
	"Alias": [
	"GNU General Public License v3.0",
	"GNU General Public License"
	]
},
{
	"URL":   "https://creativecommons.org/publicdomain/zero/1.0",
	"Name":  "CC0 1.0 Universal",
	"Alias": [
	"CC0 1.0 Universal",
	"CC0 1.0 Universal (CC0 1.0) Public Domain Dedication",
	"Creative Commons CC0 1.0 Public Domain Dedication",
	"CC0"
	]
},
{
	"URL":   "https://creativecommons.org/licenses/by-nc-sa/4.0",
	"Name":  "Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International Public License",
	"Alias": [
	"Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International Public License",
	"Attribution-NonCommercial-ShareAlike 4.0 International (CC BY-NC-SA 4.0)",
	"Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International (CC BY-NC-SA 4.0)",
	"Attribution-NonCommercial-ShareAlike 4.0 International",
	"CC-BY-NC-SA 4.0",
	"CC-BY-NC-SA"
	]
},
{
	"URL":   "https://creativecommons.org/licenses/by-nc-nd/4.0",
	"Name":  "Creative Commons Attribution-NonCommercial-NoDerivatives 4.0 International Public License",
	"Alias": [
	"Creative Commons Attribution-NonCommercial-NoDerivatives 4.0 International Public License",
	"Attribution-NonCommercial-NoDerivatives 4.0 International (CC BY-NC-ND 4.0)",
	"CC-BY-NC-ND 4.0",
	"CC-BY-NC-ND"
	]
},
{
	"URL":   "https://creativecommons.org/licenses/by-nc/4.0",
	"Name":  "Creative Commons Attribution-NonCommercial 4.0 International Public License",
	"Alias": [
	"Creative Commons Attribution-NonCommercial 4.0 International Public License",
	"Attribution-NonCommercial 4.0 International (CC BY-NC 4.0)",
	"CC BY-NC 4.0",
	"CC BY-NC"
	]
},
{
	"URL":   "https://creativecommons.org/licenses/by-sa/4.0",
	"Name":  "Creative Commons Attribution-ShareAlike 4.0 International Public License",
	"Alias": [
	"Creative Commons Attribution-ShareAlike 4.0 International Public License",
	"Attribution-ShareAlike 4.0 International (CC BY-SA 4.0)",
	"Creative Commons Attribution-ShareAlike 4.0",
	"CC BY-SA 4.0",
	"CC BY-SA"
	]
},
{
	"URL":   "https://creativecommons.org/licenses/by/4.0",
	"Name":  "Creative Commons Attribution 4.0 International Public License",
	"Alias": [
	"Creative Commons Attribution 4.0 International Public License",
	"Creative Commons Attribution 4.0 International License",
	"Attribution 4.0 International (CC BY 4.0)",
	"CC BY 4.0",
	"CC BY"
	]
}
]
`
