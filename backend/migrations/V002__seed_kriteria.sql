-- V002__seed_kriteria.sql
-- 5 kriteria awal (docs/02-perancangan-aras.md bagian 1). Bobot ini USULAN dan
-- bisa diubah Admin lewat aplikasi; total harus tepat 1,00.
INSERT INTO kriteria (kode, nama, tipe, bobot, skala_keterangan, urutan) VALUES
  ('C1', 'Penghasilan per bulan',            'cost',    0.3000, NULL, 1),
  ('C2', 'Jumlah tanggungan',                'benefit', 0.2000, NULL, 2),
  ('C3', 'Kondisi tempat tinggal',           'benefit', 0.2000,
        '{"1":"layak, milik sendiri","2":"layak, sewa/menumpang","3":"kurang layak","4":"tidak layak / tanpa tempat tinggal"}', 3),
  ('C4', 'Kondisi kesehatan / disabilitas',  'benefit', 0.1500,
        '{"1":"sehat","2":"sakit ringan","3":"sakit kronis / lansia","4":"disabilitas berat / sakit parah"}', 4),
  ('C5', 'Status pekerjaan',                 'benefit', 0.1500,
        '{"1":"tetap","2":"serabutan teratur","3":"serabutan tidak teratur","4":"tidak bekerja"}', 5);
