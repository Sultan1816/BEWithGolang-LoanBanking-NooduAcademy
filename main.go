package main

import "fmt"

// // Struct: seperti cetakan kartu, ada kolom juga
// type Produk struct {
// 	Nama string
// 	Harga int
// 	Stok
// }

// type ItemKeranjang struct {
// 	Nama string
// 	Harga int
// 	Jumlah int
// }

type MobilToyota struct {
	Nama string
	Harga int
	Stok int
}

type ItemKeranjangMobil struct {
	Nama string
	Harga int
	Jumlah int
}

// p := Produk{Nama: "Mouse Wireless Silent", Harga: 125000, Stok: 15}
// fmt.Println(p.Nama, p.Harga)	// titik = menunjuk kolom
// p.Stok = p.Stok - 2				// mengubah isi kolom

// // Slice: rak berisi banyak kartu(Struct)
// nama := []string{"Keyboard", "Mouse", "Headset". "Webcam", "Mousepad"}
// len(nama)	// 5 (jumlah sisi)
// nama[0]		// "Keyboard" (indeks mulai dari 0)

// fmt.Printf("%d. %s\n", 1, "Keyboard")

// Fungsi(mesin jus): menerima parameter(masukan), proses, keluaran(mengembalikan nilai lewat return)
func tampilkanMobil(daftar []MobilToyota) {
	for i, m := range daftar {
		fmt.Printf("%d. %s - %s (stok %d)\n", i+1, m.Nama, (m.Harga), m.Stok)
	}
}

func hitungDiskon(subtotal int) int {	// nama(parameter tipe) tipeKeluaran
	if subtotal >= 300000 {
		return subtotal * 10 / 100
	}
	return 0							// WAJIB ada
}

func bacaInput(prompt string) int {
	var input int
	fmt.Print(prompt)
	fmt.Scanln(&input)
	return input
}

func main() {
	daftarMobilToyota := []MobilToyota{
		{Nama: "Toyota Avanza", Harga: 200000000, Stok: 3},
		{Nama: "Toyota Innova", Harga: 350000000, Stok: 5},
		{Nama: "Toyota Fortuner", Harga: 500000000, Stok: 3},
		{Nama: "Toyota Alphard", Harga: 700000000, Stok: 2},
	}

	// fmt.Println(len(daftarMobilToyota))
	// fmt.Println("Daftar Mobil 0 + 3:", daftarMobilToyota[0].Nama, "&", daftarMobilToyota[3].Nama, daftarMobilToyota[0].Harga + daftarMobilToyota[3].Harga, daftarMobilToyota[0].Stok + daftarMobilToyota[3].Stok)

	// perulangan for
	fmt.Println("===== Daftar Mobil Toyota =====")
	for i := 0; i <= 3; i++ {
		fmt.Println("Daftar Mobil: ", daftarMobilToyota[i].Nama, daftarMobilToyota[i].Harga, daftarMobilToyota[i].Stok)
	}

	var keranjang []MobilToyota
	for {
		nomor := bacaInput("Pilih nomor mobil 1-3 (0 untuk keluar): ")
		if nomor == 0 {
			break
		}
		if nomor < 1 || nomor > 3 {
			fmt.Println("Nomor mobil tidak valid. Silahkan pilih nomor antara 1 - 3.")
			continue
		}

		// Mengurangi stok mobil yang dibeli
		if daftarMobilToyota[nomor-1].Stok > 0 {
			daftarMobilToyota[nomor-1].Stok --
			keranjang = append(keranjang, daftarMobilToyota[nomor-1])
			fmt.Println("Mobil berhasil ditambahkan ke keranjang.")
		}

		// Menampilkan stok mobil yang tersedia
		fmt.Println("Daftar Mobil Toyota:")
		for i, mobil := range daftarMobilToyota {
			fmt.Println(i+1, mobil.Nama, "Harga:", mobil.Harga, "Stok:", mobil.Stok)
		}
	}

	// keranjang belanja
	fmt.Println("Keranjang Belanja:")
	if len(keranjang) == 0 {
		fmt.Println("Keranjang belanja kosong.")
	} else {
		totalbayar := 0
		for _, mobil := range keranjang {
			fmt.Println(mobil.Nama, "Harga:", mobil.Harga)
			totalbayar += mobil.Harga
		}
		fmt.Println("Total Bayar:", totalbayar)
	}
}
