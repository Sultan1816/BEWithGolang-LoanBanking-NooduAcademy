package main

import "strconv"

// rupiah mengubah angka menjadi teks rupiah, misalnya 335000 → "Rp335.000".
// Fungsi ini sudah disediakan. Cara membuat fungsi sendiri dibahas di Pertemuan 2.
func rupiah(angka int) string {
	tanda := ""
	if angka < 0 {
		tanda = "-"
		angka = -angka
	}

	teks := strconv.Itoa(angka)
	hasil := ""
	for i, huruf := range teks {
		if i > 0 && (len(teks)-i)%3 == 0 {
			hasil += "."
		}
		hasil += string(huruf)
	}
	return tanda + "Rp" + hasil
}
