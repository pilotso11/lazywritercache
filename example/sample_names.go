// MIT License
//
// Copyright (c) 2023 Seth Osher
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import "math/rand"

type Name struct {
	forename string
	surname  string
}

// https://generatedata.com/generator
var (
	SampleNames = []Name{
		{"Guy", "Weaver"},
		{"Fulton", "Macdonald"},
		{"Kelsie", "Miles"},
		{"Garrett", "Cardenas"},
		{"Knox", "Dunlap"},
		{"Isaiah", "Langley"},
		{"Aquila", "Slater"},
		{"Jaime", "Navarro"},
		{"Xavier", "Weiss"},
		{"Perry", "Butler"},
		{"Jennifer", "Duncan"},
		{"Curran", "Hanson"},
		{"Malcolm", "Tyler"},
		{"Cameron", "Weaver"},
		{"Ferdinand", "Howe"},
		{"Velma", "French"},
		{"Jared", "Finch"},
		{"Ria", "Sawyer"},
		{"Alice", "Nguyen"},
		{"Octavius", "Mcconnell"},
		{"Arsenio", "Norman"},
		{"Dustin", "Walls"},
		{"Gisela", "Navarro"},
		{"Dominic", "Melton"},
		{"Callie", "Page"},
		{"Amir", "Ratliff"},
		{"Jesse", "Evans"},
		{"Jordan", "Glenn"},
		{"Rae", "Townsend"},
		{"Julian", "Goff"},
		{"Hop", "Scott"},
		{"Yoko", "Moss"},
		{"Nora", "Vang"},
		{"Amber", "Cook"},
		{"Yael", "Peck"},
		{"Adrienne", "Nash"},
		{"Wesley", "Nicholson"},
		{"Grady", "Armstrong"},
		/*		{"Byron", "Dejesus"},
				{"Demetria", "Marsh"},
				{"Andrew", "Buckner"},
				{"Cherokee", "Collier"},
				{"Guinevere", "Crawford"},
				{"Linus", "Hanson"},
				{"Driscoll", "Stokes"},
				{"Ariel", "Foreman"},
				{"Shannon", "Mcbride"},
				{"Hedwig", "Butler"},
				{"Joel", "Fuller"},
				{"Paula", "Olsen"},
				{"Samantha", "Clay"},
				{"Gannon", "Rodriguez"},
				{"Tatum", "Cole"},
				{"Imelda", "Juarez"},
				{"Willa", "Byrd"},
				{"Mannix", "Santana"},
				{"Faith", "Maddox"},
				{"Joelle", "Rollins"},
				{"Philip", "Simmons"},
				{"Kiara", "Kaufman"},
				{"Guy", "Britt"},
				{"Josiah", "Haynes"},
				{"Mira", "Pearson"},
				{"Inez", "Greer"},
				{"Kirby", "Head"},
				{"Cyrus", "Burks"},
				{"Amethyst", "Mooney"},
				{"Doris", "Meyers"},
				{"Benjamin", "Trevino"},
				{"Herrod", "Guzman"},
				{"Amela", "Medina"},
				{"Octavius", "Mckinney"},
				{"Demetria", "Whitney"},
				{"Graham", "Duncan"},
				{"Yuli", "Bullock"},
				{"Molly", "Abbott"},
				{"Cade", "Lowery"},
				{"Kimberly", "Matthews"},
				{"Ginger", "Mccoy"},
				{"Yeo", "Noble"},
				{"Zia", "Glover"},
				{"Kevyn", "Drake"},
				{"August", "Ellis"},
				{"Chaney", "Adkins"},
				{"Delilah", "Slater"},
				{"Lydia", "Parsons"},
				{"Moses", "Holder"},
				{"Ignatius", "Alston"},
				{"Rosalyn", "Mckee"},
				{"Tucker", "Henry"},
				{"Laurel", "Estes"},
				{"Anika", "Tyson"},
				{"Marsden", "White"},
				{"Igor", "Walter"},
				{"Belle", "Michael"},
				{"Solomon", "Bartlett"},
				{"Palmer", "Spencer"},
				{"Thaddeus", "Fletcher"},
				{"Darryl", "Macdonald"},
				{"Burton", "Ballard"}, */
	}

	Cities = []string{"name",
		"Lipa",
		"Kurgan",
		"Pinkafeld",
		"Kadiyivka",
		"Ternitz",
		"Izium",
		"Hofors",
		"Dublin",
		"Gijón",
		"Burhaniye",
		"Tarakan",
		"Chervonohrad",
		"Seattle",
		"San Diego",
		"Gapan",
		"Funtua",
		"Dubno",
		"Kostroma",
		"Đồng Hới",
		"Pondicherry",
		"Banjarbaru",
		"Mandai",
		"Traun",
		"Juliaca",
		"Goes",
		"Whangarei",
		"Indianapolis",
		"Wörgl",
		"Imphal",
		"Indore",
		"Abaetetuba",
		"Akhisar",
		"Tambov",
		"Kohima",
		"Ávila",
		"Cần Thơ",
		"Nicoya",
		"Elbistan",
		"Gatchina",
		"Llanelli",
		"Santa Rita",
		"Vienna",
		"Bogotá",
		"Provo",
		"Nuragus",
		"Durg",
		"Van",
		"Wieze",
		"Nova Kakhovka",
		"Los Lagos",
		"Bostaniçi",
		"Pondicherry",
		"Emalahleni",
		"Yishun",
		"Cagayan de Oro",
		"Burnie",
		"Alto del Carmen",
		"Belo Horizonte",
		"Huacho",
		"Täby",
		"Bremerhaven",
		"Walhain",
		"Nancagua",
		"Galway",
		"Jecheon",
		"Zhytomyr",
		"Gulfport",
		"Schönebeck",
		"Edremit",
		"Dereham",
		"Paya Lebar",
		"Götzis",
		"Sokoto",
		"North Waziristan",
		"Orlando",
		"Saint John",
		"Lidingo",
		"Istanbul",
		"Lillois-WitterzŽe",
		"Tehuacán",
		"Empangeni",
		"Egersund",
		"Sandnessjøen",
		"Blue Mountains",
		"Changi",
		"Sungei Kadut",
		"Bayeux",
		"Miramichi",
		"Punggol",
		"Badalona",
		"Mulhouse",
		"Kungälv",
		"Imphal",
		"Soledad de Graciano Sánchez",
		"Vienna",
		"Toruń",
		"Znamensk",
		"Orlando",
		"Banda Aceh",
		"Kaluga",
	}
)

func GetRandomName() string {
	f := rand.Intn(len(SampleNames))
	s := rand.Intn(len(SampleNames))
	return SampleNames[f].forename + " " + SampleNames[s].surname
}

func GetRandomCity() string {
	c := rand.Intn(len(Cities))
	return Cities[c]
}
