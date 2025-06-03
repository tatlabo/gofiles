import sys
import os
import sqlite3
from sqlite3 import Error

 
skipDirectories = {".git", "node_modules", "tmp", "temp", ".vscode", ".idea", "vendor", "build", "dist", "__pycache__", ",bin", ".vite"}

skipFiles = {".DS_Store", ".gitignore", ".gitattributes", ".gitmodules", "package-lock.json", "yarn.lock", "dpx", ".gitignore"}




def queryValue(cur, sql, fields=None, error=None) :
    row = queryRow(cur, sql, fields, error)
    if row is None: 
        return None
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



def folder_checker():
    ...

def stctime(fin):
    return round(os.stat(fin).st_ctime)


def drop_tables(conn, tables):

    cur = conn.cursor()
    sql_list = []
    
    for n in tables:
        sql_list.append(f'DROP TABLE IF EXISTS {n};')   

    for i in sql_list:
        cur.execute(i)

    conn.commit()



def path_to_sqdb(path, conn):

    cur = conn.cursor()
    sql_list = []
    
    path = os.path.abspath(path)

    lista = []
    for root, dirs, files in os.walk(path):
        # path = root.split(os.sep)
        lista.append(root)

    sql_list.append('CREATE TABLE IF NOT EXISTS path (id INTEGER PRIMARY KEY, name VARCHAR(256) UNIQUE);')
    sql_list.append(''' CREATE TABLE IF NOT EXISTS file (id INTEGER PRIMARY KEY, name VARCHAR(128), 
                            unixepoch INTERGER,
                            date NUMERIC,
                            path_id INTEGER, ext TEXT, ext_id INTEGER, 
                            UNIQUE(name, unixepoch, path_id),
                            FOREIGN KEY(path_id) REFERENCES path(id) ON DELETE CASCADE, 
                            FOREIGN KEY(ext_id) REFERENCES ext(id) ON DELETE CASCADE);'''   )
    sql_list.append('CREATE TABLE IF NOT EXISTS ext (id INTEGER PRIMARY KEY, ext TEXT UNIQUE);')                                       

    for i in sql_list:
        cur.execute(i)

    for folder in lista:   
        cur.execute('INSERT OR IGNORE INTO path(name) VALUES (?);', (folder,))
 
    conn.commit()


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


def update_extensins(conn):

    cur = conn.cursor()
    for row in cur.execute("SELECT DISTINCT ext FROM file;"):
        print(row[0])

    cur.execute("INSERT OR IGNORE INTO ext(ext) SELECT DISTINCT ext FROM file;")
    conn.commit()

    sql_update = ['''UPDATE OR IGNORE file SET ext_id = (SELECT ext.id FROM ext WHERE ext.ext = file.ext);''']
    sql_update.append('''UPDATE OR IGNORE file SET date = (datetime(file.unixepoch, 'unixepoch')) WHERE file.unixepoch > 0''')
    # sql_update.append('''ALTER TABLE file DROP COLUMN ext;''')
    sql_update.append('''DROP INDEX IF EXISTS file_index;''')
    sql_update.append('''CREATE INDEX file_index ON file(name);''')

    for i in sql_update:
        doExecute(cur, i)

    conn.commit()


def main():

    if len(sys.argv) == 3:
            path = sys.argv[1] # replace('\\', '/', 1)
            db_file = sys.argv[2]
        
    else:
        path = input("Enter path to index: ") # replace('\\', '/', 1)
        db_file = input("Enter db name (sqlite):") +'.sqlite'


    try:
        conn = sqlite3.connect(db_file)
    except Error as e:
        print(e)
        
    cur = conn.cursor()

    # drop_tables(conn, ['file', 'path', 'ext'])

    path_to_sqdb(path, conn)

    lista_folderow_count = cur.execute("SELECT * FROM path;").fetchall()

    folder_item_nr = input(f'Folders items: {len(lista_folderow_count)}, continue (Y)es? ').lower()


    if folder_item_nr in ('y', 't'):
        # nr = 0
        # nr_of_folders_added = 0

        for k in lista_folderow_count:
            if any(substring in k[1].replace('\\', ' ') for substring in skipDirectories):
                continue
           
            else:
                for m in filelist(k[1]):
                    f_, extension = os.path.splitext(m)
                    
                    if extension.lstrip('.') in skipFiles:
                        continue
                    else:
                        doQuery(cur, 'INSERT or IGNORE INTO file(name, path_id, unixepoch, ext) VALUES (?, ?, ?, ?);', \
                        (m, k[0], stctime(k[1]+'\\'+m), extension.lstrip('.')))

        print()
        print('Progres: ', round(k[0] / len(lista_folderow_count), 3), sep='', flush=True)
        print()

    else:
        if len(lista_folderow_count) > 0:
            print('List of folder was created, but not files.')
        else:
            print('This is the end...')
        conn.close()
        sys.exit(1)

    conn.commit()

    nr_of_folders = cur.execute("SELECT COUNT (id) FROM path;").fetchone()
    nr_of_files = cur.execute("SELECT COUNT (id) FROM file;").fetchone()

    print()
    print(f"There are {nr_of_files} files from {nr_of_folders} folders in data base.")
    print()

    update_extensins(conn)

    # sql_update.append('''CREATE INDEX file_index ON file(name);''')
    conn.close()



if __name__ =='__main__':
    main()
