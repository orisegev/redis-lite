import redis

r = redis.Redis(host='localhost', port=6379, password='123')

print(r.set('foo', 'bar'))        # True
print(r.get('foo'))               # b'bar'

r.set('session', 'xyz', ex=30)
print(r.ttl('session'))           # למשל: 29

