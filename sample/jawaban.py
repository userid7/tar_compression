class Kendaraan:
    def __init__(self):
        self.suara = "..."

    def akselarasi(self):
        print(self.suara)


class Sepeda(Kendaraan):
    def __init__(self):
        self.suara = "Swoosh"
        self.rantai = "Normal"

    def akselarasi(self):
        super().akselarasi()
        self.rantai = "Perlu perbaikan"


class Mobil(Kendaraan):
    def __init__(self):
        self.suara = "Vroom"
        self.bensin = "Penuh"

    def akselarasi(self):
        super().akselarasi()
        self.bensin = "Kosong"


if __name__ == "__main__":
    sepeda = Sepeda()
    print(sepeda.rantai)
    sepeda.akselarasi()
    print(sepeda.rantai)

    print()

    mobil = Mobil()
    print(mobil.bensin)
    mobil.akselarasi()
    print(mobil.bensin)
