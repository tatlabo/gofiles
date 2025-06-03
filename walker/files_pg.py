import sys
import os
import datetime 
from sqlalchemy import create_engine
from sqlalchemy import text

import psycopg2


credential = { 'host': 'localhost', 'port': 5432, 'database': 'go', 'user': 'golang', 'password':'golang'}


def notthisfile():
    return {}
    #{'archAEProjects', 'Cache', 'AE_Cache', 'Windows', 'tmp', '$AV_ASW', '$Windows.~WS', 'Temp' }

def notthisfolder():
    return {}
    # return {   'aecache', 'dpx', 'ocio' ,'spi1d', 'xmp', 'sha256', 'flp', 'log', 'scoreboard',    \
    #     'vmem', 'vmss', 'nvram', 'vmdk','vmsd','vmx', 'vmxf', 'vmx~'
    # }


def queryValue(cur, sql, fields=None, error=None) :
    row = queryRow(cur, sql, fields, error)
    if row is None : return None
    return row[0]

def queryRow(cur, sql, fields=None, error=None) :
    row = doQuery(cur, sql, fields)
    try:
        row = cur.fetchone()
        return row
    except Exception as e:
        if error: 
            print(error, e)
        else :
            print(e)
        return None

def doQuery(cur, sql, fields=None) :
    row = cur.execute(sql, fields)
    return row


def doExecute(cur, sql) :
    row = cur.execute(sql)
    return row


def create_connection(db_file):
    conn = None
    try:
        conn = sqlite3.connect(db_file)
        return conn
    except Error as e:
        print(e)


def filelist(dir_path):
    res = []
    if len(dir_path) > 1:
        for path in os.listdir(dir_path):
            if os.path.isfile(os.path.join(dir_path, path)):
                res.append(path)
    else:
        print('Path does not exist')
        sys.exit(1)
    return res


def directories(path):
    lista = []
    for root, dirs, files in os.walk(path):
        # path = root.split(os.sep)
        lista.append(root)
    return lista


def folder_checker():
    ...

def stctime(fin):
    return round(os.stat(fin).st_ctime)


engine = create_engine(f'postgresql+psycopg2://{credential["user"]}:{credential["password"]}@{credential["host"]}/{credential["database"]}')



def main():

    if len(sys.argv) == 2:
        path = sys.argv[1].replace('\\', '/', 1)
        
    else:
        path = 'D:/221024_Orange_XGS_Swiatlowod'
    
    lista_folderow = directories(path)

    sql_list = []

    sql_list.append('DROP TABLE IF EXISTS file')
    sql_list.append('DROP TABLE IF EXISTS path')
    sql_list.append('DROP TABLE IF EXISTS ext')
    sql_list.append('CREATE TABLE IF NOT EXISTS path (id SERIAL, name VARCHAR(256) UNIQUE, PRIMARY KEY(id))')
    sql_list.append('CREATE TABLE IF NOT EXISTS ext (id SERIAL, ext TEXT UNIQUE, PRIMARY KEY (id))')     
    sql_list.append('''CREATE TABLE IF NOT EXISTS file (id SERIAL, name VARCHAR(128), unixepoch INTEGER, date NUMERIC, ext TEXT, path_id INTEGER, FOREIGN KEY(path_id) REFERENCES path(id) ON DELETE CASCADE, ext_id INTEGER, FOREIGN KEY(ext_id) REFERENCES ext(id) ON DELETE CASCADE)''')
                                   

    with engine.connect() as conn:
        for single_sql in sql_list:
            conn.execute(text(single_sql))
        conn.commit()

        for folder in lista_folderow:   
            conn.execute(text("INSERT INTO path(name) VALUES (:x);"), [{"x": f"{folder}"}],)
        conn.commit()

        lf_count = conn.execute(text("SELECT name FROM path;"))
        conn.commit()
        
        db_folders = []
        for i in lf_count:
            db_folders.append(i.name)
        print(db_folders)

        folder_item_nr = input('continue (Y)es? ').lower()
        nr = 0
        for k in db_folders:
            if any(substring in k.name.replace('\\', ' ') for substring in notthisfolder()):
                continue
            else:
                for m in filelist(k):
                    f_, extension = os.path.splitext(m)
                    if extension.lstrip('.') in notthisfile():
                        continue
                    else:

                        conn.execute(text("INSERT INTO file(name, path_id, unixepoch, ext) VALUES (:x, :y, :t, :z);"), 
                        [{"x": m}, {"y": k}, {"t": stctime(k+'\\'+m)}, {"z": f"{extension.lstrip('.')}"}],)
                        nr += 1
            print()
            # print('OK', nr_of_folders_added)
            print()

        else:
            if len(lista_folderow_count) > 0:
                print('List of folder was created, but not files.')
            else:
                print('This is the end...')
            conn.close()
            sys.exit(1)

        conn.commit()

        print( nr )
        # print(f"{nr} files added from {nr_of_folders_added} folders")
        print()

        for row in cur.execute("SELECT DISTINCT ext FROM file;"):
            print(row[0])

        cur.execute("INSERT INTO ext(ext) SELECT DISTINCT ext FROM file;")
        conn.commit()
        cur.execute('''UPDATE file SET ext_id = (SELECT ext.id FROM ext WHERE ext.ext = file.ext);''')
        cur.execute('''UPDATE file SET date = ( datetime (unixepoch, 'unixepoch')) WHERE unixepoch > 0;''')
        cur.execute('''ALTER TABLE file DROP COLUMN ext;''')
        conn.commit()

        conn.close()




    conn.close()

if __name__ =='__main__':
    main()
