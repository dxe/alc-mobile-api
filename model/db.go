package model

import (
	"context"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

func NewDB(dsn string) *sqlx.DB {
	ctx := context.Background()

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	ping(ctx, db, time.Now().Add(time.Minute))

	log.Println("connected to database")
	// TODO(jhobbs): Init db here or in main?
	InitDatabase(db)
	return db
}

// ping repeatedly tries to ping the database (at most once per
// second) until deadline to ensure the connection is valid.
func ping(ctx context.Context, db *sqlx.DB, deadline time.Time) {
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	for {
		err := db.PingContext(ctx)
		if err == nil {
			return // success
		}
		log.Printf("database ping failed: %v", err)

		select {
		case <-ctx.Done():
			log.Fatalf("timed out trying to ping database")
		default:
		}

		time.Sleep(time.Second) // pause before trying again
	}
}

func InitDatabase(db *sqlx.DB) {
	db.MustExec(`
CREATE TABLE IF NOT EXISTS conferences (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(80) NOT NULL,
    start_date DATETIME NOT NULL,
    end_date DATETIME NOT NULL
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	conference_id INTEGER,
	name VARCHAR(200) NOT NULL DEFAULT '',
    email VARCHAR(200) NOT NULL DEFAULT '',
    device_id VARCHAR(200),
    device_name VARCHAR(200),
    platform VARCHAR(60),
    FOREIGN KEY (conference_id) REFERENCES conferences(id)
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS locations (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(200) NOT NULL,
    place_id VARCHAR(200),
    address VARCHAR(200) NOT NULL,
    city VARCHAR(100) NOT NULL,
    lat FLOAT(10,6),
    lng FLOAT(10,6)
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS events (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	conference_id INTEGER,
	name VARCHAR(200),
    description TEXT,
    start_time DATETIME NOT NULL,
    length INTEGER NOT NULL,
    location_id INTEGER,
    image_url VARCHAR(128),
    key_event TINYINT NOT NULL DEFAULT '0',
    FOREIGN KEY (conference_id) REFERENCES conferences(id),
    FOREIGN KEY (location_id) REFERENCES locations(id)
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS rsvp (
	event_id INTEGER NOT NULL,
	user_id INTEGER NOT NULL,
	attending TINYINT NOT NULL DEFAULT '0',
    PRIMARY KEY (event_id, user_id),
    FOREIGN KEY (event_id) REFERENCES events(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS info (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	title VARCHAR(200) NOT NULL,
    subtitle VARCHAR(200) NOT NULL,
    content TEXT,
    icon VARCHAR(30),
    display_order INTEGER NOT NULL
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS announcements (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	conference_id INTEGER,
	title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    icon VARCHAR(30) NOT NULL,
    created_by VARCHAR(100) NOT NULL,
    send_time DATETIME,
    sent TINYINT NOT NULL DEFAULT '0',
    FOREIGN KEY (conference_id) REFERENCES conferences(id)
)
`)

}

// WipeDatabase drops all tables in the database.
func WipeDatabase(db *sqlx.DB, flagProd bool) {
	if flagProd {
		log.Fatalln("Cannot wipe database in prod! Exiting!")
	}
	db.MustExec(`DROP TABLE IF EXISTS rsvp`)
	db.MustExec(`DROP TABLE IF EXISTS users`)
	db.MustExec(`DROP TABLE IF EXISTS events`)
	db.MustExec(`DROP TABLE IF EXISTS images`)
	db.MustExec(`DROP TABLE IF EXISTS locations`)
	db.MustExec(`DROP TABLE IF EXISTS info`)
	db.MustExec(`DROP TABLE IF EXISTS announcements`)
	db.MustExec(`DROP TABLE IF EXISTS conferences`)
}

func InsertMockData(db *sqlx.DB, flagProd bool) {
	if flagProd {
		log.Fatalln("Cannot insert mock data into db in prod! Exiting!")
	}

	db.MustExec(`
INSERT INTO conferences (id, name, start_date, end_date)
VALUES
	(1,'Animal Liberation Conference 2021','2021-09-24 00:00:00','2021-09-30 11:59:59')
`)

	db.MustExec(`
INSERT INTO locations (id, name, place_id, address, city, lat, lng)
VALUES
	(1,'Anna Head Almunae Hall','ChIJq6o6Q8mAj4ARsF7I2SLSZC4','252 2nd St','Oakland',37.794594,-122.271889),
	(2,'The Flying Falafel','ChIJvWI2gp5-hYARbzMnkKTNAng','2114 Shattuck Ave','Berkeley',37.874767,-122.268295),
	(3,'BLOC15','ChIJq6o6Q8mAj4ARsF7I2SLSZC6','254 2nd St','Oakland',37.794594,-122.271889)
`)

	db.MustExec(`
INSERT INTO announcements (id, title, message, icon, created_by, send_time, sent, conference_id)
VALUES
	(1,'consequat ut a','Vestibulum quam sapien, varius ut, blandit non, interdum in, ante. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Duis faucibus accumsan odio.','exclamation-triangle','almira@directactioneverywhere.com','2021-05-28 03:40:00',0,1),
	(2,'porttitor pede justo','Vivamus in felis eu sapien cursus vestibulum. Proin eu mi. Sa ac enim. In tempor, turpis nec euismod scelerisque, quam turpis adipiscing lorem, vitae mattis nibh ligula nec sem.','cloud','cassie@directactioneverywhere.com','2020-08-08 16:49:15',0,1),
	(3,'augue aliquam erat','Donec ut dolor. Morbi vel lectus in quam fringilla rhoncus. Mauris enim leo, rhoncus sed, vestibulum sit amet, cursus id, turpis.','newspaper-o','almira@directactioneverywhere.com','2020-09-26 05:13:03',1,1),
	(4,'vehicula condimentum curabitur','In blandit ultrices enim.','exclamation-triangle','cassie@directactioneverywhere.com','2020-11-14 07:05:22',1,1),
	(5,'at turpis donec','Quisque arcu libero, rutrum ac, lobortis vel, dapibus at, diam. Nam tristique tortor eu pede.','bus','almira@directactioneverywhere.com','2020-06-24 03:59:49',1,1),
	(6,'turpis','In hac habitasse platea dictumst.','exclamation-triangle','cassie@directactioneverywhere.com','2020-09-24 05:00:28',1,1),
	(7,'semper','In hac habitasse platea dictumst.','newspaper-o','almira@directactioneverywhere.com','2020-12-31 01:11:01',1,1),
	(8,'interdum venenatis','Morbi sem mauris, laoreet ut, rhoncus aliquet, pulvinar sed, nisl.','newspaper-o','cassie@directactioneverywhere.com','2020-09-15 09:45:15',1,1),
	(9,'at','Pellentesque viverra pede ac diam. Cras pellentesque volutpat dui.','cloud','almira@directactioneverywhere.com','2020-07-23 17:24:54',1,1),
	(10,'eu','Donec ut mauris eget massa tempor convallis. Sa neque libero, convallis eget, eleifend luctus, ultricies eu, nibh. Quisque id justo sit amet sapien dignissim vestibulum.','newspaper-o','cassie@directactioneverywhere.com','2021-03-15 06:45:41',1,1),
	(11,'Sa','In quis justo. Maecenas rhoncus aliquam lacus. Morbi quis tortor id Sa ultrices aliquet. Maecenas leo odio, condimentum id, luctus nec, molestie sed, justo.','cloud','almira@directactioneverywhere.com','2020-06-28 14:28:46',0,1),
	(12,'dui luctus','Sa tellus. In sagittis dui vel nisl. Duis ac nibh. Fusce lacus purus, aliquet at, feugiat non, pretium quis, lectus.','newspaper-o','cassie@directactioneverywhere.com','2021-03-25 21:14:40',0,1),
	(13,'dolor','In hac habitasse platea dictumst. Morbi vestibulum, velit id pretium iaculis, diam erat fermentum justo, nec condimentum neque sapien placerat ante.','bus','almira@directactioneverywhere.com','2020-06-16 11:43:15',0,1),
	(14,'platea','Donec quis orci eget orci vehicula condimentum. Curabitur in libero ut massa volutpat convallis. Morbi odio odio, elementum eu, interdum eu, tincidunt in, leo.','newspaper-o','cassie@directactioneverywhere.com','2020-06-29 20:29:27',0,1),
	(15,'eleifend pede','Pellentesque ultrices mattis odio.','newspaper-o','almira@directactioneverywhere.com','2020-08-27 16:10:39',0,1)
`)

	db.MustExec(`
INSERT INTO events (id, name, description, start_time, length, location_id, image_url, key_event, conference_id)
VALUES
	(1,'Registration','Pellentesque at nulla. Suspendisse potenti. Cras in purus eu magna vulputate luctus.\n\nCum sociis natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Vivamus vestibulum sagittis sapien. Cum sociis natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus.','2021-09-24 17:00:01',60,1,NULL,0,1),
	(2,'Event 2','Integer ac leo. Pellentesque ultrices mattis odio. Donec vitae nisi.\n\nNam ultrices, libero non mattis pulvinar, nulla pede ullamcorper augue, a suscipit nulla elit ac nulla. Sed vel enim sit amet nunc viverra dapibus. Nulla suscipit ligula in lacus.\n\nCurabitur at ipsum ac tellus semper interdum. Mauris ullamcorper purus sit amet nulla. Quisque arcu libero, rutrum ac, lobortis vel, dapibus at, diam.','2021-09-24 18:00:01',90,2,NULL,0,1),
	(3,'Event 11','Integer ac leo. Pellentesque ultrices mattis odio. Donec vitae nisi.\n\nNam ultrices, libero non mattis pulvinar, nulla pede ullamcorper augue, a suscipit nulla elit ac nulla. Sed vel enim sit amet nunc viverra dapibus. Nulla suscipit ligula in lacus.\n\nCurabitur at ipsum ac tellus semper interdum. Mauris ullamcorper purus sit amet nulla. Quisque arcu libero, rutrum ac, lobortis vel, dapibus at, diam.','2021-09-24 18:00:01',60,1,NULL,1,1),
	(4,'Event 3','Lorem ipsum dolor sit amet, consectetuer adipiscing elit. Proin risus. Praesent lectus.\n\nVestibulum quam sapien, varius ut, blandit non, interdum in, ante. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia Curae; Duis faucibus accumsan odio. Curabitur convallis.','2021-09-24 20:00:01',60,2,NULL,0,1),
	(5,'Event 4','Cum sociis natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Vivamus vestibulum sagittis sapien. Cum sociis natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus.','2021-09-24 23:00:01',75,3,NULL,1,1),
	(6,'Event 5','Proin interdum mauris non ligula pellentesque ultrices. Phasellus id sapien in sapien iaculis congue. Vivamus metus arcu, adipiscing molestie, hendrerit at, vulputate vitae, nisl.\n\nAenean lectus. Pellentesque eget nunc. Donec quis orci eget orci vehicula condimentum.\n\nCurabitur in libero ut massa volutpat convallis. Morbi odio odio, elementum eu, interdum eu, tincidunt in, leo. Maecenas pulvinar lobortis est.','2021-09-25 01:30:01',90,1,NULL,0,1),
	(7,'Event 6','Sed sagittis. Nam congue, risus semper porta volutpat, quam pede lobortis ligula, sit amet eleifend pede libero quis orci. Nullam molestie nibh in lectus.','2021-09-25 17:00:01',0.75,1,NULL,0,1),
	(8,'Event 7','Vestibulum ac est lacinia nisi venenatis tristique. Fusce congue, diam id ornare imperdiet, sapien urna pretium nisl, ut volutpat sapien arcu sed augue. Aliquam erat volutpat.\n\nIn congue. Etiam justo. Etiam pretium iaculis justo.','2021-09-25 18:00:01',120,1,NULL,0,1),
	(9,'Event 8','Phasellus sit amet erat. Nulla tempus. Vivamus in felis eu sapien cursus vestibulum.','2021-09-25 20:00:01',60,1,NULL,1,1),
	(10,'Event 9','Suspendisse potenti. In eleifend quam a odio. In hac habitasse platea dictumst.','2021-09-25 23:00:01',180,1,NULL,0,1),
	(11,'Event 10','Fusce consequat. Nulla nisl. Nunc nisl.\n\nDuis bibendum, felis sed interdum venenatis, turpis enim blandit mi, in porttitor pede justo eu massa. Donec dapibus. Duis at velit eu est congue elementum.','2021-09-26 01:30:01',45,1,NULL,1,1)
`)

	db.MustExec(`
INSERT INTO info (id, title, subtitle, content, icon, display_order)
VALUES
	(1,'FAQ','Get answers to commonly asked questions.','<p><strong>Title</strong><br/>Text</p><p><strong>Title</strong><br/>Text</p>','question',1),
	(2,'Community Agreements','Help us maintain a safe and empowering space.','<p><strong>Title</strong><br/>Text</p><p><strong>Title</strong><br/>Text</p>','handshake-o',2),
	(3,'Contact Us','Reach the organizers if you have any questions or concerns.','<p><strong>Title</strong><br/>Text</p><p><strong>Title</strong><br/>Text</p>','envelope-o',3),
	(4,'Chants & Lyrics',"Unsure of what's being said or sang? Follow along here.",'<p><strong>Title</strong><br/>Text</p><p><strong>Title</strong><br/>Text</p>','microphone',4)
`)

}
