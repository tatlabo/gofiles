import sys
import os
import csv
# import psycopg2
# from hidden import pglog
import sqlite3

import mysql.connector



def stctime(fin):
    return round(os.stat(fin).st_ctime)

def check(s, status):
    t = (s, status)
    if status in ['', '0', 0]:
        t = (s, 'TRUE')
    elif int(status) == 8:
        t = ('NULL', 'FALSE')
    else:
        t = ('NULL', 'TRUE')

    return t

def insert_daily_kd(path, table):

    headers ="""
        code_id, term, tmax, tmin, tavg, tmin_ground, tmin_ground_status,
        precipitation, precipitation_status, precipitation_type,
        snow_cover, snow_cover_status"""

    daily_sql_list = []

    with open(path, mode='r', encoding='latin1') as csvfile:
        csvreader = csv.reader(csvfile)
        for row in csvreader:
            code_id, station, yyyy, mm, dd, \
            tmax, tmax_status, tmin, tmin_status, taverage, taverage_status, \
            tmin_ground, tmin_ground_status, precipitation, precipitation_status, \
            precipitation_type, snow_cover, snow_cover_status = tuple(row)


            tmax, tmax_status = check(tmax, tmax_status)
            tmin, tmin_status = check(tmin, tmin_status)
            taverage, taverage_status = check(taverage, taverage_status)
            snow_cover, snow_cover_status = check( snow_cover, snow_cover_status )
            precipitation, precipitation_status = check(precipitation, precipitation_status)
            tmin_ground, tmin_ground_status = check(tmin_ground, tmin_ground_status)


            term = f"{yyyy}-{mm}-{dd}"

            # daily_sql_list.append(f""" --sql
            #     INSERT INTO station (code, station) VALUES ({code_id}, '{station}')
            #     ON CONFLICT (code) DO NOTHING
            # ;""")

            item = (f"""
                INSERT INTO {table} ({headers}) VALUES
                ({code_id}, STR_TO_DATE('{term}', '%Y-%m-%d'), {tmax}, {tmin}, {taverage}, {tmin_ground}, {tmin_ground_status},
                {precipitation}, {precipitation_status}, '{precipitation_type}', {snow_cover}, {snow_cover_status})
                ON DUPLICATE KEY UPDATE
                tmax = VALUES(tmax),
                tmin = VALUES(tmin),
                tavg = VALUES(tavg),
                tmin_ground = VALUES(tmin_ground),
                tmin_ground_status = VALUES(tmin_ground_status),
                precipitation = VALUES(precipitation),
                precipitation_status = VALUES(precipitation_status),
                precipitation_type = VALUES(precipitation_type),
                snow_cover = VALUES(snow_cover),
                snow_cover_status = VALUES(snow_cover_status)
            ;""")

            daily_sql_list.append(item)

        return daily_sql_list

def insert_daily_kdt(path, table):

    headers ="""
        code_id, term, taverage, taverage_status,
        humidity, humidity_status, wind_average, wind_average_status,
        oktan, oktan_status"""

    daily_sql_list = []

    with open(path, mode='r', encoding='latin1') as csvfile:

        csvreader = csv.reader(csvfile)
        for row in csvreader:

            code_id, station, yyyy, mm, dd, taverage, taverage_status, \
            humidity, humidity_status, wind_average, wind_average_status, \
            oktan, oktan_status = tuple(row)

            taverage, taverage_status = check(taverage, taverage_status)
            humidity, humidity_status = check(humidity, humidity_status)
            wind_average, wind_average_status = check(wind_average, wind_average_status)
            oktan, oktan_status = check(oktan , oktan_status)

            term = f"{yyyy}-{mm}-{dd}"

            # daily_sql_list.append(f""" --sql
            #     INSERT INTO station (code, station) VALUES ({code_id}, '{station}')
            #     ON CONFLICT (code) DO NOTHING;""")

            item = f"""
                INSERT INTO {table} ({headers}) VALUES
                ({code_id}, STR_TO_DATE('{term}', '%Y-%m-%d'), {taverage}, {taverage_status},
                {humidity}, {humidity_status}, {wind_average}, {wind_average_status},
                {oktan}, {oktan_status})
                ON DUPLICATE KEY UPDATE
                taverage = VALUES(taverage),
                taverage_status = VALUES(taverage_status),
                humidity = VALUES(humidity),
                humidity_status = VALUES(humidity_status),
                wind_average = VALUES(wind_average),
                wind_average_status = VALUES(wind_average_status),
                oktan = VALUES(oktan),
                oktan_status = VALUES(oktan_status);
            """
            daily_sql_list.append(item)



        return daily_sql_list


def csv_from_sqlite(db_path):


    try:
        conn_sqlite = sqlite3.connect(db_path)
    except:
        print("sth wrong")

    cur_sqlite = conn_sqlite.cursor()

    cur_sqlite.execute('''SELECT path.name||'/'||file.name, status FROM path JOIN file ON file.path_id = path.id WHERE file.status < 1 OR file.status IS NULL;''')
    get_csv_paths = cur_sqlite.fetchall()

    csv_paths = []
    for k in get_csv_paths:
        csv_paths.append(k[0])


    try:
        conn = mysql.connector.connect(
            host='tatlabo.mysql.eu.pythonanywhere-services.com',
            database = 'tatlabo$meteo',
            user='tatlabo',
            password='$meteo@tatlabo'
        )
    except mysql.connector.Error as error:
       print(error)

    cur = conn.cursor()



    prefix = 'meteo'

    for path in csv_paths:

        print(path)

        i = path.split('/')[-1]

        print(i)

        if i.startswith("k_d_t_"):
            daily_sql_list = insert_daily_kdt(path, prefix)
        elif i.startswith("k_d_"):
            daily_sql_list = insert_daily_kd(path, prefix)
        else:
            print("nie ma poprawnego „csv”.")
            sys.exit()


        for day in daily_sql_list:
            cur.execute(day)
        conn.commit()

        _01 = f"""--sql
        UPDATE file SET status = 10 WHERE name = '{i}';"""
        cur_sqlite.execute(_01)
        conn_sqlite.commit()

        _02 = f"""--sql
            SELECT status FROM file WHERE name = '{i}';"""
        j = cur_sqlite.execute(_02).fetchone()
        print(f"{i}, status: {j[0]}")

    try:
        cur.execute(f'SELECT COUNT(id) FROM {prefix};')
        _count = cur.fetchone()[0]
        print(f'There is: {_count} items in table „{prefix}”.')
    except:
        print(f'Table {prefix} does not exist.')

    conn.close()
    conn_sqlite.close()




def main():
    db_path = './csv.sqlite'
    csv_from_sqlite(db_path)


if __name__=="__main__":
    main()

