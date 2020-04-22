import unittest
import main

class Tests(unittest.TestCase):

    def test_parse(self):
        # int
        self.assertEqual(main.parseMsg(b'123'), [123])
        # string
        self.assertEqual(main.parseMsg(b'"123"'), [b"123"])
        # string with embedded "
        self.assertEqual(main.parseMsg(b'"a\\"b"'), [b'a"b'])
        # array
        self.assertEqual(main.parseMsg(b'"123",233,"a\nb"'), [b'123', 233, b'a\nb'])
    def test_encode(self):
        # int
        self.assertEqual(main.encodeMsg([123]), b"123")
        # string
        self.assertEqual(main.encodeMsg(["123"]), b'"123"')
        # string with embedded "
        self.assertEqual(main.encodeMsg(['a"b']), b'"a\\"b"')
        # array
        self.assertEqual(main.encodeMsg(['123', 233, 'a\nb']), b'"123",233,"a\nb"')

'''
    def test_isupper(self):
        self.assertTrue('FOO'.isupper())
        self.assertFalse('Foo'.isupper())

    def test_split(self):
        s = 'hello world'
        self.assertEqual(s.split(), ['hello', 'world'])
        # check that s.split fails when the separator is not a string
        with self.assertRaises(TypeError):
            s.split(2)
'''

if __name__ == '__main__':
    unittest.main()
