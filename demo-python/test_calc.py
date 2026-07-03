# demo-python/test_calc.py
import pytest
from calc import add, subtract

def test_add():
    assert add(1, 2) == 3

def test_add_negative():
    assert add(-1, 1) == 0

def test_subtract():
    assert subtract(5, 3) == 2
